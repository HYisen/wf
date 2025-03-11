package main

import (
	"context"
	"fmt"
	. "github.com/hyisen/wf"
	"log"
	"net/http"
	"time"
)

type Request struct {
	Data []byte
	Path string
}

func WithPath(data []byte, path string) (any, error) {
	return Request{
		Data: data,
		Path: path,
	}, nil
}

func main() {
	SetTimeout(100 * time.Millisecond) // 100ms is long enough for local dev
	handler := NewClosureHandler(
		// matcher is under which circumstance the handler would be assigned for dispatch.
		Exact(http.MethodGet, "/echo"),
		// parser is how the handler would extract data from payload and path in request.
		WithPath,
		// handler is how the handler maps Request to Response, or error.
		func(_ context.Context, req any) (rsp any, codedError *CodedError) {
			r := req.(Request)
			msg := fmt.Sprintf("path=%s\ndata=%v", r.Path, r.Data)
			return msg, nil
		},
		// formatter is how handler encodes response value to bytes.
		func(output any) (data []byte, err error) {
			return []byte(output.(string)), nil
		},
		// contentType is that in HTTP response Header.
		"text/plain")
	web := NewWeb(false, handler)
	if err := http.ListenAndServe("localhost:8080", web); err != nil {
		log.Fatal(err)
	}
}
