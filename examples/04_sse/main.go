package main

import (
	"context"
	"github.com/hyisen/wf"
	"log"
	"log/slog"
	"net/http"
	"time"
)

const (
	// one item every second, change timeoutSeconds and itemSize to decide how would it ends.
	timeoutSeconds = 10
	itemSize       = 6
)

func main() {
	wf.SetTimeout(time.Second)
	handler := wf.NewServerSentEventsHandler(
		wf.Exact(http.MethodPost, "/events"),
		wf.ParseEmpty,
		func(ctx context.Context, req any) (<-chan wf.MessageEvent, *wf.CodedError) {
			ch := make(chan wf.MessageEvent)
			ticker := time.NewTicker(time.Second)
			slog.Info("start")
			go pass(ctx, ch, ticker)
			return ch, nil
		},
	)
	handler.Timeout = timeoutSeconds * time.Second // override global timeout
	web := wf.NewWeb(false, handler)
	if err := http.ListenAndServe("localhost:8080", web); err != nil {
		log.Fatal(err)
	}
}

func pass(ctx context.Context, ch chan<- wf.MessageEvent, ticker *time.Ticker) {
	defer close(ch)
	for range itemSize {
		select {
		case <-ticker.C:
			slog.Info("tick")
			ch <- wf.MessageEvent{
				TypeOptional: "",
				Lines:        []string{time.Now().String()},
			}
		case <-ctx.Done():
			slog.Info("stop ticker as ctx done", "reason", ctx.Err())
			return
		}
	}
}
