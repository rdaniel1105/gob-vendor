package main

import (
	"context"

	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/collector"
	"github.com/rdaniel1105/gob-vendor/logger"
)

func main() {
	client, err := client.New()
	if err != nil {
		panic(err)
	}

	fetcher, err := collector.NewFetcher("Suministro de Bienes y/o Servicios", "Recepción de Ofertas", "Compra Menor", client, logger.New("collector"))
	if err != nil {
		panic(err)
	}

	err = fetcher.Execute(context.Background())
	if err != nil {
		panic(err)
	}
}
