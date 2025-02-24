package main

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/collector"
	"github.com/rdaniel1105/gob-vendor/logger"
)

var (
	maxPaginationConcurrency = runtime.NumCPU() / 2

	sleepTime = 10 * time.Second
)

func processAllPages(ctx context.Context, pages int64) error {
	errChan := make(chan error)
	defer close(errChan)

	semaphore := make(chan struct{}, maxPaginationConcurrency)
	wg := sync.WaitGroup{}

	for i := int64(2); i <= pages; i++ {
		time.Sleep(sleepTime)

		wg.Add(1)
		go func(pageNum int64) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			logger := logger.New("collector")
			client, err := client.New()
			if err != nil {
				errChan <- err
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
			}

			err = fetcher.Execute(ctx, pageNum)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()

	for err := range errChan {
		if err != nil {
			slog.Error("error_processing_page", "error", err)
		}
	}

	return nil
}

func main() {
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

	slog.Info("pages", "pages", fetcher.GetPages())

	err = processAllPages(ctx, fetcher.GetPages())
	if err != nil {
		slog.Error("error_processing_pages", "error", err)
	}
}
