package collector

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/rdaniel1105/gob-vendor/storage"
)

const (
	jumpLine = "\n"
)

var (
	maxConcurrentSenders = 10
	chunkSize            = 30
)

type chunkState struct {
	counter int
	builder strings.Builder
}

func detailsChunkSender(detailSender chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	chunkState := chunkState{
		counter: 0,
		builder: strings.Builder{},
	}

	limiter := make(chan struct{}, maxConcurrentSenders)

	for detail := range detailSender {

		chunkState.builder.WriteString(detail)
		chunkState.builder.WriteString(jumpLine)

		chunkState.counter++

		if chunkState.counter == chunkSize {
			sendChunk(getChunk(chunkState), wg, limiter)

			chunkState.reset()
		}
	}

	if chunkState.counter > 0 {
		sendChunk(getChunk(chunkState), wg, limiter) // no need to reset state, because it's the last chunk
	}
}

func getChunk(chunkState chunkState) string {
	chunk := chunkState.builder.String()

	return chunk
}

func sendChunk(chunk string, wg *sync.WaitGroup, limiter chan struct{}) {
	wg.Add(1)
	limiter <- struct{}{}

	go bulkUpload(chunk, wg, limiter)
}

func (cs *chunkState) reset() {
	cs.counter = 0
	cs.builder.Reset()
}

func bulkUpload(data string, wg *sync.WaitGroup, limiter chan struct{}) {
	defer wg.Done()
	defer func() { <-limiter }()

	if err := storage.NewZincSearchClient(zincSearchURL, admin, password).Save(context.Background(), data); err != nil {
		log.Printf("bulk upload error: %v", err)

		return
	}
}
