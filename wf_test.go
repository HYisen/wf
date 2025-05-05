package wf

import (
	"cmp"
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
)

func OrElseV0(optional HaveOptionalTimeout, fallback time.Duration) time.Duration {
	ret := fallback
	if optional.TimeoutOptional() != 0 {
		ret = optional.TimeoutOptional()
	}
	return ret
}

func OrElseV1(optional HaveOptionalTimeout, fallback time.Duration) time.Duration {
	return cmp.Or(optional.TimeoutOptional(), fallback)
}

func generateTimeoutConfigs(emptyRatio float64, size int) []*TimeoutConfig {
	random := rand.New(rand.NewPCG(0, 17))
	var ret []*TimeoutConfig
	for range size {
		var tc TimeoutConfig
		if random.Float64() > emptyRatio {
			// I just don't want not repeatable rand.N.
			tc.Timeout = time.Duration(random.IntN(int(1000 * time.Millisecond)))
		}
		ret = append(ret, &tc)
	}
	return ret
}

func BenchmarkOrElse(b *testing.B) {
	inputs := generateTimeoutConfigs(0.5, 1000)
	b.Run("v0", func(b *testing.B) {
		for b.Loop() {
			for _, input := range inputs {
				OrElseV0(input, timeout)
			}
		}
	})
	b.Run("v1", func(b *testing.B) {
		for b.Loop() {
			for _, input := range inputs {
				OrElseV1(input, timeout)
			}
		}
	})
}

func TestOrElse(t *testing.T) {
	inputs := generateTimeoutConfigs(0.1, 1000)
	for _, input := range inputs {
		want := OrElseV0(input, timeout)
		got := OrElseV1(input, timeout)
		if want != got {
			t.Errorf("v0 and v1 got different result %v and %v", want, got)
		}
	}
}

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

func NewEchoHandler(path string, timeout time.Duration, handler HandleFunc) Handler {
	h := NewClosureHandler(
		Exact(http.MethodGet, path),
		func(data []byte, _ string) (any, error) {
			return data, nil
		},
		handler,
		func(output any) (data []byte, err error) {
			return output.([]byte), nil
		},
		"text/plain")
	h.Timeout = timeout
	return h
}

const LevelNever = slog.LevelError + 1

func TestInboundTimeout(t *testing.T) {
	old := slog.SetLogLoggerLevel(LevelNever)
	defer slog.SetLogLoggerLevel(old)
	path := "/echo"
	timeout := 50 * time.Millisecond
	handler := func(_ context.Context, req any) (rsp any, codedError *CodedError) {
		return req, nil
	}
	server := httptest.NewServer(NewWeb(false, NewEchoHandler(path, timeout, handler)))
	defer server.Close()
	req, err := http.NewRequest(http.MethodGet, server.URL+path, NewSlowReader([]byte("12345"), timeout))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("want 500 status code, got %d", resp.StatusCode)
	}
}

type SlowReader struct {
	data     []byte
	cursor   int
	interval time.Duration
}

func NewSlowReader(data []byte, cost time.Duration) *SlowReader {
	intervalMS := int(math.Ceil(float64(cost.Milliseconds()) / float64(len(data)-1)))
	return &SlowReader{
		data:     data,
		cursor:   0,
		interval: time.Duration(intervalMS) * time.Millisecond,
	}
}

func (r *SlowReader) Read(p []byte) (n int, err error) {
	if r.cursor >= len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.cursor]
	r.cursor++
	time.Sleep(r.interval)
	return 1, nil
}

func TestOutboundTimeout(t *testing.T) {
	path := "/echo-very-slow"
	duration := 50 * time.Millisecond
	handler := func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
		time.Sleep(duration*2 + writeDeadlineExtension)
		return req, nil
	}
	server := httptest.NewServer(NewWeb(false, NewEchoHandler(path, duration, handler)))
	defer server.Close()
	req, err := http.NewRequest(http.MethodGet, server.URL+path, strings.NewReader("12345"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.Client().Do(req)

	if err == nil || !errors.Is(err, io.EOF) {
		t.Errorf("want err EOF, got %v", err)
	}
}

func TestHandleTimeout(t *testing.T) {
	old := slog.SetLogLoggerLevel(LevelNever)
	defer slog.SetLogLoggerLevel(old)
	path := "/echo-slow"
	timeout := 50 * time.Millisecond
	handler := func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
		time.Sleep(timeout + time.Millisecond)
		deadline, ok := ctx.Deadline()
		if ok && deadline.Before(time.Now()) {
			return nil, NewCodedError(http.StatusInternalServerError, errors.New("timeout"))
		}
		return req, nil
	}
	server := httptest.NewServer(NewWeb(false, NewEchoHandler(path, timeout, handler)))
	defer server.Close()
	req, err := http.NewRequest(http.MethodGet, server.URL+path, strings.NewReader("12345"))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode/100 == 2 {
		t.Errorf("want not 2XX status code, got %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	keyword := "timeout"
	if !strings.Contains(string(data), keyword) {
		t.Errorf("want response with keyword %q, got %v", keyword, string(data))
	}
}
