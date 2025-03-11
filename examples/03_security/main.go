package main

import (
	"context"
	"errors"
	. "github.com/hyisen/wf"
	"log"
	"net/http"
)

func valid(token string) bool {
	return token == "top_secret"
}

func main() {
	handler := NewClosureHandler(
		Exact(http.MethodPost, "/v1/vital"),
		ParseEmpty,
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			token := DetachToken(ctx)
			if token == "" {
				return nil, NewCodedError(http.StatusUnauthorized, errors.New("need token"))
			}
			if !valid(token) {
				return nil, NewCodedErrorf(http.StatusForbidden, "invalid token %s", token)
			}
			return nil, nil
		},
		FormatEmpty,
		"text/plain",
	)
	web := NewWeb(false, handler)
	if err := http.ListenAndServe("localhost:8080", web); err != nil {
		log.Fatal(err)
	}
}
