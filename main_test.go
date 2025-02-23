package main

import (
	"context"
	"testing"

	"github.com/rdaniel1105/gob-vendor/client"
	"github.com/rdaniel1105/gob-vendor/collector"
	"github.com/rdaniel1105/gob-vendor/logger"
)

func TestMain(t *testing.T) {
	client, err := client.New()
	if err != nil {
		t.Fatal(err)
	}

	fetcher, err := collector.NewFetcher("Suministro de Bienes y/o Servicios", "Recepción de Ofertas", "Compra Menor", client, logger.New("collector"))
	if err != nil {
		t.Fatal(err)
	}

	fetcher.Execute(context.Background())
}
