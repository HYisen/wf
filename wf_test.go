package wf

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
)

func TestResourceWithIDs(t *testing.T) {
	type args struct {
		method string
		parts  []string
	}
	one := args{
		method: http.MethodGet,
		parts:  []string{"users", "", "items", ""},
	}
	tests := []struct {
		name   string
		args   args
		method string
		rawURL string
		match  bool
		ids    []int
	}{
		{"happy path", one, http.MethodGet, "https://a.com/users/123/items/456", true, []int{123, 456}},
		{"method", one, http.MethodPost, "https://a.com/users/123/items/456", false, nil},
		{"prefix slash", one, http.MethodGet, "users/123/items/456", true, []int{123, 456}},
		{"suffix slash", one, http.MethodGet, "https://a.com/users/123/items/456/", true, []int{123, 456}},
		{"suffix more", one, http.MethodGet, "https://a.com/users/123/items/456/desc", false, nil},
		{"not number id", one, http.MethodGet, "https://a.com/users/abc/items/456", false, nil},
		{"bad name", one, http.MethodGet, "https://a.com/user/123/items/456", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mf, pf := ResourceWithIDs(tt.args.method, tt.args.parts)

			u, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatalf("parse url failed: %v", err)
			}

			ok := mf(&http.Request{Method: tt.method, URL: u})
			if ok != tt.match {
				t.Errorf("match got %t want %t", ok, tt.match)
			}

			if ok {
				data, err := pf(nil, u.Path)
				if err != nil {
					t.Errorf("parse not nil error: %v", err)
				}
				if !slices.Equal(data.([]int), tt.ids) {
					t.Errorf("parse got %v, want %v", data, tt.ids)
				}
			}
		})
	}
}
