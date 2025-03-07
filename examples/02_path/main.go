package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	. "wf"
)

type Request struct {
	ID   int
	Body string
}

type Response struct {
	PathID int    `json:"id"`
	Body   string `json:"body"`
}

func main() {
	simple := &ClosureHandler{
		Matcher: ResourceWithID(http.MethodDelete, "/v1/widgets/", ""),
		Parser:  PathIDParser(""),
		Handler: func(_ context.Context, req any) (rsp any, codedError *CodedError) {
			return &Response{
				PathID: req.(int),
				Body:   "NA",
			}, nil
		},
		Formatter:   json.Marshal,
		ContentType: JSONContentType,
	}

	suffix := "/content"
	parser := PathIDParser(suffix)
	comprehensive := &ClosureHandler{
		Matcher: ResourceWithID(http.MethodPost, "/v1/items/", suffix),
		Parser: func(data []byte, path string) (any, error) {
			id, err := parser(nil, path)
			if err != nil {
				return nil, err
			}
			return &Request{
				ID:   id.(int),
				Body: string(data),
			}, nil
		},
		Handler: func(_ context.Context, req any) (rsp any, codedError *CodedError) {
			r := req.(*Request)
			return &Response{
				PathID: r.ID,
				Body:   r.Body,
			}, nil
		},
		Formatter:   json.Marshal,
		ContentType: JSONContentType,
	}

	web := NewWeb(false, simple, comprehensive)
	if err := http.ListenAndServe("localhost:8080", web); err != nil {
		log.Fatal(err)
	}
}
