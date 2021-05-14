package client

import (
	"context"
	"net/http"
	"net/http/httptrace"

	"github.com/dop251/goja"
	jsHTTP "go.k6.io/k6/js/modules/k6/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type TracingClient struct {
	http *jsHTTP.HTTP
}

type HTTPResponse struct {
	*jsHTTP.Response
	TraceID string
}

type HttpFunc func(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error)

func New() *TracingClient {
	return &TracingClient{
		http: &jsHTTP.HTTP{},
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
	ctx, _, span := startTraceAndSpan(ctx, spanName)
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
