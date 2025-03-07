package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	. "wf"
)

func valid(token string) bool {
	return token == "top_secret"
}

func main() {
	handler := &ClosureHandler{
		Matcher: Exact(http.MethodPost, "/v1/vital"),
		Parser:  ParseEmpty,
		Handler: func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			token := DetachToken(ctx)
			if token == "" {
				return nil, NewCodedError(http.StatusUnauthorized, errors.New("need token"))
			}
			if !valid(token) {
				return nil, NewCodedErrorf(http.StatusForbidden, "invalid token %s", token)
			}
			return nil, nil
		},
		Formatter:   FormatEmpty,
		ContentType: "text/plain",
	}
	web := NewWeb(false, handler)
	if err := http.ListenAndServe("localhost:8080", web); err != nil {
		log.Fatal(err)
	}
}
