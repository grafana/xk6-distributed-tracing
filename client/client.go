package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
	"unsafe"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/modules"
	k6HTTP "go.k6.io/k6/js/modules/k6/http"
	"go.k6.io/k6/lib"
)

type TracingClient struct {
	vu          modules.VU
	httpRequest HttpRequestFunc

	endpoint  string
	testRunID int64

	httpClient *http.Client
}

type HTTPResponse struct {
	*k6HTTP.Response
	TraceID string
}

type (
	HttpRequestFunc func(method string, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error)
	HttpFunc        func(ctx context.Context, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error)
)

func New(vu modules.VU, requestFunc HttpRequestFunc, endpoint string, testRunID int64) *TracingClient {
	return &TracingClient{
		httpRequest: requestFunc,
		vu:          vu,
		endpoint:    endpoint,
		testRunID:   testRunID,
		httpClient:  &http.Client{},
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

func (c *TracingClient) WithTrace(fn HttpFunc, spanName string, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	ctx := c.vu.Context()

	traceID, _, _ := EncodeTraceID(TraceID{
		Prefix:            K6Prefix,
		Code:              K6Code_Cloud,
		UnixTimestampNano: uint64(time.Now().UnixNano()) / uint64(time.Millisecond)})

	h, _ := GenerateHeaderBasedOnPropagator(PropagatorW3C, "123")

	headers := map[string][]string{}
	for key, header := range h {
		headers[key] = header
	}

	vm := goja.New()
	val := vm.ToValue(map[string]map[string][]string{
		"headers": headers,
	})

	args = append(args, val)
	res, e := fn(ctx, url, args...)

	var scenario string
	scenarioState := lib.GetScenarioState(ctx)

	// In case we do requests on the setup/teardown steps
	if scenarioState == nil {
		scenario = ""
	} else {
		scenario = scenarioState.Name
	}

	r := []*Request{{
		TestRunID:         c.testRunID,
		StartTimeUnixNano: uint64(time.Now().UnixNano()) - uint64(res.Timings.Duration*1000000),
		EndTimeUnixNano:   uint64(time.Now().UnixNano()),
		Group:             "group",
		Scenario:          scenario,
		TraceID:           traceID,
		HTTPUrl:           "hola",
		HTTPMethod:        "adios",
		HTTPStatus:        int64(123),
	}}

	payload, err := json.Marshal(RequestBatch{
		SizeBytes: int64(unsafe.Sizeof(r)),
		Count:     int64(len(r)),
		Requests:  r,
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal request metadata")
	}

	rq, _ := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(payload))
	rq.Header.Add("X-Scope-OrgID", "123")
	_, err = c.httpClient.Do(rq)
	if err != nil {
		logrus.WithError(err).Error("Failed to send request metadata")
	}

	return &HTTPResponse{Response: res, TraceID: traceID}, e
}
