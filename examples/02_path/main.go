package main

import (
	"context"
	"encoding/json"
	. "github.com/hyisen/wf"
	"log"
	"net/http"
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
	simple := NewClosureHandler(
		ResourceWithID(http.MethodDelete, "/v1/widgets/", ""),
		PathIDParser(""),
		func(_ context.Context, req any) (rsp any, codedError *CodedError) {
			return &Response{
				PathID: req.(int),
				Body:   "NA",
			}, nil
		},
		json.Marshal,
		JSONContentType,
	)

	suffix := "/content"
	parser := PathIDParser(suffix)
	comprehensive := NewClosureHandler(
		ResourceWithID(http.MethodPost, "/v1/items/", suffix),
		func(data []byte, path string) (any, error) {
			id, err := parser(nil, path)
			if err != nil {
				return nil, err
			}
			return &Request{
				ID:   id.(int),
				Body: string(data),
			}, nil
		},
		func(_ context.Context, req any) (rsp any, codedError *CodedError) {
			r := req.(*Request)
			return &Response{
				PathID: r.ID,
				Body:   r.Body,
			}, nil
		},
		json.Marshal,
		JSONContentType,
	)

	mf, pf := ResourceWithIDs(http.MethodGet, []string{"v1", "users", "", "items", ""})
	complicated := NewClosureHandler(mf, pf, func(_ context.Context, req any) (rsp any, codedError *CodedError) {
		ids := req.([]int)
		return ids, nil
	}, json.Marshal, JSONContentType)

	combined := NewClosureHandler(
		MatchAll(
			Exact(http.MethodGet, "/v1/ask"),
			HasQuery("q", "this_is_a_question"),
		),
		ParseEmpty,
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return "The answer is 42.", nil
		},
		json.Marshal,
		JSONContentType,
	)

	web := NewWeb(false, simple, comprehensive, complicated, combined)
	if err := http.ListenAndServe("localhost:8080", web); err != nil {
		log.Fatal(err)
	}
}
