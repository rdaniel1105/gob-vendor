package logger

import (
	"errors"
	"io"
	"log/slog"
)

var (
	ErrEmptyService = errors.New("service name is required")
)

// Logger data handler
type Logger struct {
	ClientID       string
	UserID         string
	ServiceName    string
	HasProbability bool
	Probability    float64
	Output         io.Writer
}

func New(serviceName string) *slog.Logger {
	if serviceName == "" {
		panic(ErrEmptyService)
	}

	output := io.Discard

	return slog.New(slog.NewJSONHandler(output, nil))
}
