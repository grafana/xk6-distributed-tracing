package tracing

import (
	"bytes"
	"context"
	"net/http"
	"time"
	"unsafe"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
	jsHTTP "go.k6.io/k6/js/modules/k6/http"
	"go.k6.io/k6/lib"
	"google.golang.org/protobuf/proto"
)

type TracingClient struct {
	http       *jsHTTP.HTTP
	httpClient *http.Client
	Backend    string
	TestRunID  int
	OrgID      string
	APIKey     string
}

type HTTPResponse struct {
	*jsHTTP.Response
	TraceID string
}

type HttpFunc func(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error)

func New(backend string, testRunID int, orgID string, apiKey string) *TracingClient {
	return &TracingClient{
		http:       &jsHTTP.HTTP{},
		httpClient: &http.Client{},
		Backend:    backend,
		TestRunID:  testRunID,
		OrgID:      orgID,
		APIKey:     apiKey,
	}
}

func (c *TracingClient) Get(ctx context.Context, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Get, "HTTP GET", ctx, url, args...)
}

func (c *TracingClient) Post(ctx context.Context, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Post, "HTTP POST", ctx, url, args...)
}

func (c *TracingClient) Put(ctx context.Context, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Put, "HTTP PUT", ctx, url, args...)
}

func (c *TracingClient) Del(ctx context.Context, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Del, "HTTP DEL", ctx, url, args...)
}

func (c *TracingClient) Head(ctx context.Context, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Head, "HTTP HEAD", ctx, url, args...)
}

func (c *TracingClient) Patch(ctx context.Context, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Patch, "HTTP PATCH", ctx, url, args...)
}

func (c *TracingClient) Options(ctx context.Context, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Options, "HTTP OPTIONS", ctx, url, args...)
}

func (c *TracingClient) WithTrace(fn HttpFunc, spanName string, ctx context.Context, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	traceID, _, _ := EncodeTraceID(TraceID{
		Prefix:            K6Prefix,
		Code:              K6Code_Cloud,
		UnixTimestampNano: uint64(time.Now().UnixNano()) / uint64(time.Millisecond)})

	vm := goja.New()

	h, _ := GenerateHeaderBasedOnPropagator(PropagatorJaeger, traceID)

	headers := map[string][]string{}
	for key, header := range h {
		headers[key] = header
	}

	val := vm.ToValue(map[string]map[string][]string{
		"headers": headers,
	})
	args = append(args, val)

	res, err := fn(ctx, url, args...)

	if c.Backend != "" {
		var scenario string
		globalState := lib.GetState(ctx)
		scenarioState := lib.GetScenarioState(ctx)

		// In case we do requests on the setup/teardown steps
		if scenarioState == nil {
			scenario = ""
		} else {
			scenario = scenarioState.Name
		}

		r := []*Request{{
			TestRunID:         int64(c.TestRunID),
			StartTimeUnixNano: uint64(time.Now().UnixNano()) - uint64(res.Timings.Duration*1000000),
			EndTimeUnixNano:   uint64(time.Now().UnixNano()),
			Group:             globalState.Tags["group"],
			Scenario:          scenario,
			TraceID:           traceID,
			HTTPUrl:           res.Request.URL,
			HTTPMethod:        res.Request.Method,
			HTTPStatus:        int64(res.Status),
		}}

		md := &RequestBatch{
			SizeBytes: int64(unsafe.Sizeof(r)),
			Count:     int64(len(r)),
			Requests:  r,
		}

		mm, err := proto.Marshal(md)
		if err != nil {
			logrus.WithError(err).Error("Failed to marshal request metadata")
		}

		rq, _ := http.NewRequest("POST", c.Backend, bytes.NewBuffer(mm))
		rq.SetBasicAuth(c.OrgID, c.APIKey)
		ra, err := c.httpClient.Do(rq)
		if err != nil {
			logrus.WithError(err).Error("Failed to send request metadata")
		}
		if ra.StatusCode != 200 {
			logrus.WithError(err).Error("Failed to send request metadata", ra.StatusCode)
		}
	}

	return &HTTPResponse{Response: res, TraceID: traceID}, err
}
