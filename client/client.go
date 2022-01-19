package client

import (
	"context"
	"net/http"
	"net/http/httptrace"

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
	http *k6HTTP.HTTP
}

type HTTPResponse struct {
	*k6HTTP.Response
	TraceID string
}

type HttpFunc func(ctx context.Context, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error)

func New(vu modules.VU) *TracingClient {
	return &TracingClient{
		http: &k6HTTP.HTTP{},
		vu:   vu,
	}
}

func (c *TracingClient) Get(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Get, "HTTP GET", url, args...)
}

func (c *TracingClient) Post(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Post, "HTTP POST", url, args...)
}

func (c *TracingClient) Put(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Put, "HTTP PUT", url, args...)
}

func (c *TracingClient) Del(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Del, "HTTP DEL", url, args...)
}

func (c *TracingClient) Head(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Head, "HTTP HEAD", url, args...)
}

func (c *TracingClient) Patch(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Patch, "HTTP PATCH", url, args...)
}

func (c *TracingClient) Options(url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	return c.WithTrace(c.http.Options, "HTTP OPTIONS", url, args...)
}

func (c *TracingClient) WithTrace(fn HttpFunc, spanName string, url goja.Value, args ...goja.Value) (*HTTPResponse, error) {
	ctx, _, span := startTraceAndSpan(c.vu.Context(), spanName)
	defer span.End()

	id := span.SpanContext().TraceID().String()

	ctx, val := getTraceHeadersArg(ctx)

	args = append(args, val)
	res, err := fn(ctx, url, args...)

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
