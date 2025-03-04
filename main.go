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

func processAllPages(ctx context.Context, pages int64) error {
	errChan := make(chan error)
	semaphore := make(chan struct{}, maxPaginationConcurrency)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

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

				err = fetcher.Execute(ctx, pageNum)
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
		adquisitionType,
		adquisitionStage,
		adquisitionCategory,
		client,
		logger.New("collector"),
	)
	if err != nil {
		panic(err)
	}

	err = fetcher.Execute(ctx, 1)
	if err != nil {
		slog.Error("error_fetching_data", "error", err)
	}

	err = processAllPages(ctx, fetcher.GetPages())
	if err != nil {
		slog.Error("error_processing_pages", "error", err)
	}
}
