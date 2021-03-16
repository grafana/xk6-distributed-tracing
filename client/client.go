package client

import (
	"context"
	"github.com/dop251/goja"
	jsHTTP "github.com/loadimpact/k6/js/modules/k6/http"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/http/httptrace"
)

type TracingClient struct {
	http *jsHTTP.HTTP
}

type HttpFunc func(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error)

func New() *TracingClient {
	return &TracingClient{
		http: &jsHTTP.HTTP{},
	}
}

func (c *TracingClient) Get(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error) {
	return c.WithTrace(c.http.Get, ctx, url, args...)
}

func (c *TracingClient) Post(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error) {
	return c.WithTrace(c.http.Post, ctx, url, args...)
}


func (c *TracingClient) Put(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error) {
	return c.WithTrace(c.http.Put, ctx, url, args...)
}


func (c *TracingClient) Del(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error) {
	return c.WithTrace(c.http.Del, ctx, url, args...)
}


func (c *TracingClient) Head(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error) {
	return c.WithTrace(c.http.Head, ctx, url, args...)
}


func (c *TracingClient) Patch(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error) {
	return c.WithTrace(c.http.Patch, ctx, url, args...)
}


func (c *TracingClient) Options(ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error) {
	return c.WithTrace(c.http.Options, ctx, url, args...)
}


func (c *TracingClient) WithTrace(fn HttpFunc, ctx context.Context, url goja.Value, args ...goja.Value) (*jsHTTP.Response, error) {
	ctx, _, span := startTraceAndSpan(ctx)
	defer span.End()

	id := span.SpanContext().TraceID.String()
	logrus.WithField("trace-id", id).Info("Starting trace")

	ctx, val := getTraceHeadersArg(ctx)
	
	args = append(args, val)
	res, err := fn(ctx, url, args...)
  
	// TODO: extract the textmap from the response
	return res, err
}

func startTraceAndSpan(ctx context.Context) (context.Context, trace.Tracer, trace.Span){
	trace := otel.Tracer("http/makerequest")
	ctx, span := trace.Start(ctx, "make-request")
	return ctx, trace, span
}

func getTraceHeadersArg(ctx context.Context) (context.Context, goja.Value) {
	vm := goja.New()

	ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx))

	h := http.Header{}
	otel.GetTextMapPropagator().Inject(ctx, h)

	headers := map[string][]string {}
	for key, header := range h {
		headers[key] = header
	}

	val := vm.ToValue(map[string]map[string][]string {
		"headers": headers,
	})

	return ctx, val
}