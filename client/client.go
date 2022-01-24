package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"strings"

	"github.com/dop251/goja"
	"go.k6.io/k6/js/modules"
	k6HTTP "go.k6.io/k6/js/modules/k6/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type TracingClient struct {
	vu   modules.VU
	http *k6HTTP.Client
}

type HTTPResponse struct {
	*k6HTTP.Response
	TraceID string
}

type HttpFunc func(ctx context.Context, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error)

func New(vu modules.VU) *TracingClient {
	return &TracingClient{
		http: &k6HTTP.Client{},
		vu:   vu,
	}
}

func (c *TracingClient) Get(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(http.MethodGet, url, args...)
}

func (c *TracingClient) Post(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(http.MethodPost, url, args...)
}

func (c *TracingClient) Put(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(http.MethodPut, url, args...)
}

func (c *TracingClient) Del(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(http.MethodDelete, url, args...)
}

func (c *TracingClient) Head(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(http.MethodHead, url, args...)
}

func (c *TracingClient) Patch(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(http.MethodPatch, url, args...)
}

func (c *TracingClient) Options(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(http.MethodOptions, url, args...)
}

func (c *TracingClient) WithTrace(method string, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	spanName := fmt.Sprintf("HTTP %s", strings.ToUpper(method))

	ctx, _, span := startTraceAndSpan(c.vu.Context(), spanName)
	defer span.End()

	id := span.SpanContext().TraceID().String()

	_, val := getTraceHeadersArg(ctx)

	// TODO: fix the case if the params[headers] are presented
	args = append(args, val)
	res, err := c.http.Request(method, url, args...)

	span.SetAttributes(attribute.String("http.method", res.Request.Method), attribute.Int("http.status_code", res.Response.Status), attribute.String("http.url", res.Request.URL))
	// TODO: extract the textmap from the response
	return &HTTPResponse{Response: res, TraceID: id}, err
}

func startTraceAndSpan(ctx context.Context, name string) (context.Context, trace.Tracer, trace.Span) {
	trace := otel.Tracer("xk6/http")
	ctx, span := trace.Start(ctx, name)
	return ctx, trace, span
}

func getTraceHeadersArg(ctx context.Context) (context.Context, goja.Value) {
	vm := goja.New()

	ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx))

	h := http.Header{}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(h))

	headers := map[string][]string{}
	for key, header := range h {
		headers[key] = header
	}

	val := vm.ToValue(map[string]map[string][]string{
		"headers": headers,
	})

	return ctx, val
}
