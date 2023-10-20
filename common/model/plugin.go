package model

import (
	"context"
	"time"
)

type Parser interface {
	GetLogFile() (string, error)
	GetLabels() map[string]string
	Process(ctx context.Context, lines <-chan string, sends chan<- Send, errCh chan<- error)
}

type Send struct {
	Timestamp time.Time
	Data      map[string]interface{}
}
