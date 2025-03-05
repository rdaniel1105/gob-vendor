package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/collector"
	"github.com/rdaniel1105/gob-vendor/logger"
)

var (
	maxPaginationConcurrency = runtime.NumCPU() / 2

	sleepTime              = 10 * time.Second
	longerSleepTime        = 25 * time.Second
	pagesBeforeLongerSleep = 20

	errGoDotenvLoad = errors.New("error loading .env file")
)

func processAllPages(ctx context.Context, pages int64, initialState collector.SessionState) error {
	stateChan := make(chan collector.SessionState, 50)

	stateChan <- initialState

	errChan := make(chan error, pages)
	semaphore := make(chan struct{}, maxPaginationConcurrency)
	wg := sync.WaitGroup{}

	var currentState collector.SessionState
	var stateMu sync.Mutex

	go func() {
		for state := range stateChan {
			stateMu.Lock()
			currentState = state
			stateMu.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(stateChan)

		for i := int64(2); i <= pages; i++ {
			wg.Add(1)
			go func(pageNum int64) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				logger := logger.New("collector")
				client, err := client.New()
				if err != nil {
					errChan <- err
					return
				}

				fetcher, err := collector.NewFetcher(
					collector.PaginationEventTarget,
					fmt.Sprintf(collector.PaginationEventArgument, pageNum),
					pageNum,
					collector.AdquisitionTypeServicesAndGoods,
					collector.AdquisitionStageReceivedOffersType,
					collector.AdquisitionCategoryMinorPurchaseType,
					client,
					logger,
				)
				if err != nil {
					errChan <- err
					return
				}

				fetcher.SessionStateChan = stateChan

				stateMu.Lock()
				state := currentState
				stateMu.Unlock()

				fetcher.ViewState = state.ViewState
				fetcher.ViewStateGenerator = state.ViewStateGenerator
				fetcher.EventValidation = state.EventValidation

				err = fetcher.Execute(ctx)
				if err != nil {
					errChan <- fmt.Errorf("error fetching page %d: %w", pageNum, err)
				}
			}(i)

			if (i-1)%int64(pagesBeforeLongerSleep) == 0 {
				time.Sleep(longerSleepTime)
			} else {
				time.Sleep(sleepTime)
			}
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			slog.Error("error_processing_page", "error", err)
		}
	}

	return nil
}

func initEnv() {
	err := godotenv.Load()
	if err != nil {
		panic(errGoDotenvLoad)
	}
}

func main() {
	initEnv()

	client, err := client.New()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	adquisitionType := collector.AdquisitionTypeServicesAndGoods
	adquisitionStage := collector.AdquisitionStageReceivedOffersType
	adquisitionCategory := collector.AdquisitionCategoryMinorPurchaseType

	fetcher, err := collector.NewFetcher(
		"",
		"",
		1,
		adquisitionType,
		adquisitionStage,
		adquisitionCategory,
		client,
		logger.New("collector"),
	)
	if err != nil {
		panic(err)
	}

	err = fetcher.Execute(ctx)
	if err != nil {
		slog.Error("error_fetching_data", "error", err)
		return
	}

	initialState := collector.SessionState{
		ViewState:          fetcher.ViewState,
		ViewStateGenerator: fetcher.ViewStateGenerator,
		EventValidation:    fetcher.EventValidation,
	}

	err = processAllPages(ctx, fetcher.GetPages(), initialState)
	if err != nil {
		slog.Error("error_processing_pages", "error", err)
	}
}
