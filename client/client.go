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
	vu          modules.VU
	httpRequest httpRequestFunc
}

type HTTPResponse struct {
	*k6HTTP.Response
	TraceID string
}

type (
	httpRequestFunc func(method string, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error)
	HttpFunc        func(ctx context.Context, url goja.Value, args ...goja.Value) (*k6HTTP.Response, error)
)

func New(vu modules.VU) *TracingClient {
	r := k6HTTP.New().NewModuleInstance(vu).Exports().Default.(*goja.Object).Get("request")
	var requestFunc httpRequestFunc
	err := vu.Runtime().ExportTo(r, &requestFunc)
	if err != nil {
		panic(err)
	}
	return &TracingClient{
		httpRequest: requestFunc,
		vu:          vu,
	}
}

func requestToHttpFunc(method string, request httpRequestFunc) HttpFunc {
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
