package main

import (
	"context"
	"encoding/json"
	"fmt"
	. "github.com/hyisen/wf"
	"log"
	"net/http"
	"reflect"
	"time"
)

type Request struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Response struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	whole := NewJSONHandler(
		Exact(http.MethodPost, "/v1/whole"),
		reflect.TypeOf(Request{}),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			r := req.(*Request)
			msg := fmt.Sprintf("[whole]%+v", r)
			return Response{Message: msg, Timestamp: time.Now()}, nil
		},
	)
	semi := NewClosureHandler(
		Exact(http.MethodPost, "/v1/semi"),
		func(data []byte, path string) (any, error) {
			var req Request
			if err := json.Unmarshal(data, &req); err != nil {
				return nil, err
			}
			// WATCH OUT, it's pointer return type!
			return &req, nil
		},
		func(_ context.Context, req any) (rsp any, codedError *CodedError) {
			r := req.(*Request)
			msg := fmt.Sprintf("[semi]%+v", r)
			return Response{Message: msg, Timestamp: time.Now()}, nil
		},
		json.Marshal,
		JSONContentType,
	)
	web := NewWeb(false, whole, semi)
	if err := http.ListenAndServe("localhost:8080", web); err != nil {
		log.Fatal(err)
	}
}
