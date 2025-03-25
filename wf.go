package wf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type CodedError struct {
	Code int
	Err  error
}

func NewCodedError(code int, err error) *CodedError {
	return &CodedError{Code: code, Err: err}
}

func NewCodedErrorf(code int, format string, a ...any) *CodedError {
	return &CodedError{Code: code, Err: fmt.Errorf(format, a...)}
}

func (e CodedError) Error() string {
	return fmt.Sprintf("%d: %v", e.Code, e.Err.Error())
}

type Handler interface {
	CanMatch
	CanParse
	CanHandle
	CanResponse
}

type CanMatch interface {
	Match(req *http.Request) bool
}

type CanParse interface {
	Parse(data []byte, path string) (any, error)
}

type HandleOutputType any

type CanHandle interface {
	Handle(ctx context.Context, req any) (HandleOutputType, *CodedError)
}

type CanResponse interface {
	// Response does not return err, as we can not respond with error if Response fails.
	// Now that the only thing we can do is log, we can log inside rather than pass out and do it outside.
	Response(output HandleOutputType, writer http.ResponseWriter)
}

type CanFormat interface {
	Format(output any) (data []byte, err error)
}

type HasResponseContentType interface {
	ResponseContentType() string // could use http.DetectContentType as default, which finds JSON as text/plain.
}

type MatchFunc func(req *http.Request) bool

func MatchAll(criteria ...MatchFunc) MatchFunc {
	return func(req *http.Request) bool {
		for _, criterion := range criteria {
			if !criterion(req) {
				return false
			}
		}
		return true
	}
}

func Exact(method string, path string) MatchFunc {
	return func(req *http.Request) bool {
		return req.URL.Path == path && req.Method == method
	}
}

func HasQuery(key string, value string) MatchFunc {
	return func(req *http.Request) bool {
		return req.URL.Query().Get(key) == value
	}
}

func ResourceWithID(method string, pathPrefixWithTailSlash string, pathSuffixWithHeadSlashNullable string) MatchFunc {
	return func(req *http.Request) bool {
		if req.Method != method {
			return false
		}
		s, found := strings.CutPrefix(req.URL.Path, pathPrefixWithTailSlash)
		if !found {
			return false
		}
		if pathSuffixWithHeadSlashNullable != "" {
			s, found = strings.CutSuffix(s, pathSuffixWithHeadSlashNullable)
			if !found {
				return false
			}
		}
		if _, err := strconv.Atoi(s); err != nil {
			return false
		}
		return true
	}
}

func ResourceWithIDs(method string, parts []string) (MatchFunc, ParseFunc) {
	mh := func(req *http.Request) bool {
		if req.Method != method {
			return false
		}
		// As I tested, whether Path contains prefix or suffix slash could depend on whether rawURL have them.
		// To avoid difference on their existence, I trim before split.
		subs := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		if len(subs) != len(parts) {
			return false
		}
		for i, part := range parts {
			if part != "" {
				if part != subs[i] {
					return false
				}
			} else {
				if _, err := strconv.Atoi(subs[i]); err != nil {
					return false
				}
			}
		}
		return true
	}
	pf := func(_ []byte, path string) (req any, err error) {
		// The implementation of pf is highly bound with mh.
		// We don't handle situations that won't happen. e.g.,
		// len(subs) != len(parts) or Atoi failure is asserted in mh.
		var ret []int
		subs := strings.Split(strings.Trim(path, "/"), "/")
		for i, part := range parts {
			if part != "" {
				continue
			}
			num, _ := strconv.Atoi(subs[i])
			ret = append(ret, num)
		}
		return ret, nil
	}
	return mh, pf
}

type HandleFunc func(ctx context.Context, req any) (rsp any, codedError *CodedError)

type closureMatcherAndParser struct {
	matcher MatchFunc
	parser  ParseFunc
}

func (c *closureMatcherAndParser) Match(req *http.Request) bool {
	return c.matcher(req)
}

func (c *closureMatcherAndParser) Parse(data []byte, path string) (any, error) {
	return c.parser(data, path)
}

// ClosureHandler is something that implements Handler with closures.
type ClosureHandler struct {
	closureMatcherAndParser
	handler     HandleFunc
	formatter   func(output any) (data []byte, err error)
	contentType string
}

func NewClosureHandler(
	matcher MatchFunc,
	parser ParseFunc,
	handler HandleFunc,
	formatter func(output any) (data []byte, err error),
	contentType string,
) *ClosureHandler {
	return &ClosureHandler{closureMatcherAndParser: closureMatcherAndParser{
		matcher: matcher,
		parser:  parser,
	}, handler: handler, formatter: formatter, contentType: contentType}
}

func (ch *ClosureHandler) Response(output HandleOutputType, writer http.ResponseWriter) {
	outputData, err := ch.Format(output)
	if err != nil {
		slog.Error("unexpected failure on marshal", "err", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", ch.ResponseContentType())
	_, _ = writer.Write(outputData)
}

const JSONContentType = "application/json; charset=utf-8"

func NewJSONHandler(matcher MatchFunc, requestType reflect.Type, handler HandleFunc) *ClosureHandler {
	return &ClosureHandler{
		closureMatcherAndParser: closureMatcherAndParser{
			matcher: matcher,
			parser:  JSONParser(requestType),
		},
		handler:     handler,
		formatter:   json.Marshal,
		contentType: JSONContentType,
	}
}

func NewServerSentEventsHandler(matcher MatchFunc, parser ParseFunc, handler StreamGenerator) *ServerSentEventsHandler {
	return &ServerSentEventsHandler{
		closureMatcherAndParser: closureMatcherAndParser{
			matcher: matcher,
			parser:  parser,
		},
		handler: handler,
	}
}

type StreamGenerator func(ctx context.Context, req any) (ch <-chan MessageEvent, codedError *CodedError)

type ServerSentEventsHandler struct {
	closureMatcherAndParser
	handler StreamGenerator
}

func (h *ServerSentEventsHandler) Handle(ctx context.Context, req any) (HandleOutputType, *CodedError) {
	return h.handler(ctx, req)
}

func (h *ServerSentEventsHandler) Response(output HandleOutputType, writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	ch := output.(<-chan MessageEvent)
	rc := http.NewResponseController(writer)
	for me := range ch {
		// I could, but I don't record write failure, because I haven't.
		if me.TypeOptional != "" {
			_, _ = fmt.Fprintf(writer, "event: %s\n", me.TypeOptional)
		}
		for _, line := range me.Lines {
			_, _ = fmt.Fprintf(writer, "data: %s\n", line)
		}
		_, _ = fmt.Fprintln(writer)
		if err := rc.Flush(); err != nil {
			slog.Error("unexpected failure on flush", "err", err)
			return
		}
	}
}

// MessageEvent represents a Server-Sent-Event.
// While transmitting, if TypeOptional is empty string "",
// a typed event would be generated, otherwise unnamed message.
// While transmitting, each item of Lines would gain prefix `data: `,
// and every item must not include LF.
//
// According to the spec, when client parse MessageEvent, should concatenate lines,
// inserting a newline character between each one. Trailing newlines are removed.
//
// ref https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
// ref https://html.spec.whatwg.org/multipage/server-sent-events.html#dispatchMessage
type MessageEvent struct {
	TypeOptional string
	Lines        []string
}

// Empty types used on JSONParser indicates that no data and shall use ParseEmpty.
type Empty struct {
}

func (ch *ClosureHandler) Handle(
	ctx context.Context,
	req any,
) (rsp HandleOutputType, codedError *CodedError) {
	return ch.handler(ctx, req)
}

func (ch *ClosureHandler) Format(output any) (data []byte, err error) {
	return ch.formatter(output)
}

func (ch *ClosureHandler) ResponseContentType() string {
	return ch.contentType
}

// Web is a helper to implements http.Handler as mux.
// There was a Handler[RequestType,ResponseType] design,
// which is good as guaranteed type consistency between its methods,
// but failed as it's []any, not []Handler[any,any] that accepts Handler[One,Two],
// and in runtime, the interface conversion from Handler[any,any] to Handler[One,Two] failed.
// Once I drop the type info, it can not come back even through cast.
// The best performance strategy could be a code generator, which is complicated to implements.
// Or just put the dirty transform work together as it was, which causes a lot of redundancy.
type Web struct {
	handlers  []Handler
	allowCORS bool
}

func NewWeb(allowCORS bool, handlers ...Handler) *Web {
	return &Web{handlers: handlers, allowCORS: allowCORS}
}

var timeout = 1000 * time.Millisecond

// SetTimeout setups the timeout of ctx created and passed by the framework.
// A default value of 1000 ms would be used without any explicit invoke of this function.
// Once ServeHTTP, which could be an instance of Web being passed to http.ListenAndServe, will NOT take effect.
// Better to used just before the start network listening action.
func SetTimeout(duration time.Duration) {
	timeout = duration
}

var serverContextCreator = func() (ctx context.Context, cancel context.CancelFunc) {
	threshold := timeout
	cause := fmt.Errorf("handler exceed timeout %v", threshold)
	return context.WithTimeoutCause(context.Background(), threshold, cause)
}

func (w *Web) findHandler(req *http.Request) Handler {
	// Maybe a Trie when it's more complicated and the performance difference matters.
	for _, h := range w.handlers {
		if h.Match(req) {
			return h
		}
	}
	return nil
}

var allowedMethods = strings.Join([]string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete}, ",")
var allowedHeaders = strings.Join([]string{"Content-Type", "Token"}, ",")

// ServeHTTP implements that in interface.
func (w *Web) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if w.allowCORS {
		writer.Header().Set("Access-Control-Allow-Origin", request.Header.Get("Origin"))
		if request.Method == http.MethodOptions {
			slog.Warn("allow cors", "req", request)
			writer.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			writer.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			writer.Header().Set("Access-Control-Max-Age", "3600") // I just love the 1hr duration.
			writer.WriteHeader(http.StatusAccepted)
			return
		}
	}

	h := w.findHandler(request)
	if h == nil {
		writer.WriteHeader(http.StatusNotAcceptable)
		slog.Warn("unmatched request", "req", request)
		_, _ = writer.Write([]byte(fmt.Sprintf("unsupported request on %v %v", request.Method, request.URL)))
		return
	}

	inputData, err := io.ReadAll(request.Body)
	if err != nil {
		// What if it's client's fault? Maybe warn rather than error?
		slog.Error("unexpected failure on read", "err", err, "req", request)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	input, err := h.Parse(inputData, request.URL.Path)
	if err != nil {
		slog.Warn("bad input format", "err", err, "req", request)
		writer.WriteHeader(http.StatusBadRequest)
		// Let it go when can not send the optional error info to client, which could be their problem.
		_, _ = writer.Write([]byte(fmt.Sprintf("can not parse req %v as %v", request, err)))
		return
	}

	ctx, cancel := serverContextCreator()
	defer cancel()
	ctx = AttachToken(ctx, request.Header.Get("Token"))
	output, e := h.Handle(ctx, input)
	if e != nil {
		if IsUserFault(e.Code) {
			slog.Warn("resp " + e.Error())
		} else {
			slog.Error("resp " + e.Error())
		}
		writer.WriteHeader(e.Code)
		_, _ = writer.Write([]byte(e.Err.Error()))
		return
	}
	h.Response(output, writer)
}

func IsUserFault(httpStatusCode int) bool {
	return httpStatusCode/100 == 4
}

const ctxTokenKey = "token"

func AttachToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, ctxTokenKey, token)
}

func DetachToken(ctx context.Context) string {
	return ctx.Value(ctxTokenKey).(string)
}

type ParseFunc func(data []byte, path string) (req any, err error)

func JSONParser(clazz reflect.Type) ParseFunc {
	if clazz == reflect.TypeOf(Empty{}) {
		return ParseEmpty
	}
	return func(data []byte, _ string) (any, error) {
		value := reflect.New(clazz)
		if err := json.Unmarshal(data, value.Interface()); err != nil {
			return value, err
		}
		return value.Interface(), nil
	}
}

func PathIDParser(pathSuffixWithHeadSlashNullable string) ParseFunc {
	return func(_ []byte, path string) (any, error) {
		if pathSuffixWithHeadSlashNullable != "" {
			rest, found := strings.CutSuffix(path, pathSuffixWithHeadSlashNullable)
			if !found {
				return 0, fmt.Errorf("no suffix %v in path %v", pathSuffixWithHeadSlashNullable, path)
			}
			path = rest
		}
		// The CanMatch shall have guaranteed a valid number here. So we can skip validation here.
		str := path[strings.LastIndexByte(path, '/')+1:]
		num, _ := strconv.Atoi(str)
		return num, nil
	}
}

func ParseEmpty(_ []byte, _ string) (any, error) {
	return nil, nil
}

func FormatEmpty(_ any) (data []byte, err error) {
	return nil, nil
}
