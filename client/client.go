package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dop251/goja"
	"go.k6.io/k6/js/modules"
	k6HTTP "go.k6.io/k6/js/modules/k6/http"
	"go.k6.io/k6/metrics"
)

type Options struct {
	Propagator string
}

type TracingClient struct {
	vu          modules.VU
	httpRequest HttpRequestFunc

	options Options
}

type HTTPResponse struct {
	*k6HTTP.Response `js:"-"`
	TraceID          string
}

type (
	HttpRequestFunc func(method string, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error)
	HttpFunc        func(ctx context.Context, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error)
)

func New(vu modules.VU, requestFunc HttpRequestFunc, options Options) *TracingClient {
	return &TracingClient{
		httpRequest: requestFunc,
		vu:          vu,
		options:     options,
	}
}

func requestToHttpFunc(method string, request HttpRequestFunc) HttpFunc {
	return func(ctx context.Context, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error) {
		return request(method, url, args...)
	}
}

func (c *TracingClient) Get(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	args = append([]goja.Value{goja.Null()}, args...)
	return c.WithTrace(requestToHttpFunc(http.MethodGet, c.httpRequest), "HTTP GET", url, args...)
}

func (c *TracingClient) Post(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(requestToHttpFunc(http.MethodPost, c.httpRequest), "HTTP POST", url, args...)
}

func (c *TracingClient) Put(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(requestToHttpFunc(http.MethodPut, c.httpRequest), "HTTP PUT", url, args...)
}

func (c *TracingClient) Del(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(requestToHttpFunc(http.MethodDelete, c.httpRequest), "HTTP DEL", url, args...)
}

func (c *TracingClient) Head(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(requestToHttpFunc(http.MethodHead, c.httpRequest), "HTTP HEAD", url, args...)
}

func (c *TracingClient) Patch(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(requestToHttpFunc(http.MethodPatch, c.httpRequest), "HTTP PATCH", url, args...)
}

func (c *TracingClient) Options(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(requestToHttpFunc(http.MethodOptions, c.httpRequest), "HTTP OPTIONS", url, args...)
}

func isNilly(val goja.Value) bool {
	return val == nil || goja.IsNull(val) || goja.IsUndefined(val)
}

func (c *TracingClient) WithTrace(fn HttpFunc, spanName string, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	state := c.vu.State()
	if state == nil {
		return nil, fmt.Errorf("HTTP requests can only be made in the VU context")
	}

	traceID, _, err := EncodeTraceID(TraceID{
		Prefix:            K6Prefix,
		Code:              K6Code_Cloud,
		UnixTimestampNano: uint64(time.Now().UnixNano()) / uint64(time.Millisecond),
	})
	if err != nil {
		return nil, err
	}

	tracingHeaders, err := GenerateHeaderBasedOnPropagator(c.options.Propagator, traceID)
	if err != nil {
		return nil, err
	}

	// This makes sure that the tracing header will always be added correctly to
	// the HTTP request headers, whether they were explicitly specified by the
	// user in the script or not.
	//
	// First we make sure to either get the existing request params, or create
	// them from scratch if they were not specified:
	rt := c.vu.Runtime()
	var params *goja.Object
	if len(args) < 2 {
		params = rt.NewObject()
		if len(args) == 0 {
			args = []goja.Value{goja.Null(), params}
		} else {
			args = append(args, params)
		}
	} else {
		jsParams := args[1]
		if isNilly(jsParams) {
			params = rt.NewObject()
			args[1] = params
		} else {
			params = jsParams.ToObject(rt)
		}
	}
	// Then we either augment the existing params.headers or create them:
	var headers *goja.Object
	if jsHeaders := params.Get("headers"); isNilly(jsHeaders) {
		headers = rt.NewObject()
		params.Set("headers", headers)
	} else {
		headers = jsHeaders.ToObject(rt)
	}
	for key, val := range tracingHeaders {
		headers.Set(key, val)
	}

	// TODO: set span_id as well as some other metadata?
	state.Tags.Modify(func(tagsAndMeta *metrics.TagsAndMeta) {
		tagsAndMeta.SetMetadata("trace_id", traceID)
	})
	defer state.Tags.Modify(func(tagsAndMeta *metrics.TagsAndMeta) {
		tagsAndMeta.DeleteMetadata("trace_id")
	})

	// This calls the actual request() function from k6/http with our augmented arguments
	res, e := fn(c.vu.Context(), url, args...)

	return &HTTPResponse{Response: res, TraceID: traceID}, e
}
