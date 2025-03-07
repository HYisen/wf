package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	. "wf"
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
	handler := &ClosureHandler{
		// Matcher is under which circumstance the handler would be assigned for dispatch.
		Matcher: Exact(http.MethodGet, "/echo"),
		// Parser is how the handler would extract data from payload and path in request.
		Parser: WithPath,
		// Handler is how the handler maps Request to Response, or error.
		Handler: func(_ context.Context, req any) (rsp any, codedError *CodedError) {
			r := req.(Request)
			msg := fmt.Sprintf("path=%s\ndata=%v", r.Path, r.Data)
			return msg, nil
		},
		// Formatter is how handler encodes response value to bytes.
		Formatter: func(output any) (data []byte, err error) {
			return []byte(output.(string)), nil
		},
		// ContentType is that in HTTP response Header.
		ContentType: "text/plain",
	}
	web := NewWeb(false, handler)
	if err := http.ListenAndServe("localhost:8080", web); err != nil {
		log.Fatal(err)
	}
}
