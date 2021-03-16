package client

import (
	"context"
	"github.com/dop251/goja"
	"github.com/loadimpact/k6/js/modules/k6/http"
	"github.com/loadimpact/k6/lib/consts"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	http2 "net/http"
	"net/http/httptrace"
	"go.opentelemetry.io/otel"
	exportjaeger "go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"os"
)

type TracingClient struct {
	http *http.HTTP
}

type HttpFunc func(ctx context.Context, url goja.Value, args ...goja.Value) (*http.Response, error)

func New() *TracingClient {
	SetupJaeger()

	return &TracingClient{
		http: &http.HTTP{},
	}
}

func SetupJaeger() {
	jaegerEndpoint, ok := os.LookupEnv("JAEGER_ENDPOINT")
	if !ok {
		jaegerEndpoint = "http://localhost:14268/api/traces"
	}
	_, err := exportjaeger.InstallNewPipeline(
		exportjaeger.WithCollectorEndpoint(jaegerEndpoint),
		exportjaeger.WithProcess(exportjaeger.Process{
			ServiceName: "k6",
			Tags: []label.KeyValue{
				label.String("exporter", "jaeger"),
				label.String("k6.version", consts.Version),
			},
		}),
		exportjaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	if err != nil {
		logrus.WithError(err).Error("Error while starting the Jaeger exporter pipeline")
	} else {
		logrus.Info("Jaeger exporter configured")
	}
}

func startTraceAndSpan(ctx context.Context) (context.Context, trace.Tracer, trace.Span){
	trace := otel.Tracer("http/makerequest")
	ctx, span := trace.Start(ctx, "make-request")
	return ctx, trace, span
}

func getTraceHeadersArg(ctx context.Context) (context.Context, goja.Value) {
	vm := goja.New()

	ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx))

	h := http2.Header{}
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

func (c *TracingClient) Get(ctx context.Context, url goja.Value, args ...goja.Value) (*http.Response, error) {
	return c.WithTrace(c.http.Get, ctx, url, args...)
}

func (c *TracingClient) Post(ctx context.Context, url goja.Value, args ...goja.Value) (*http.Response, error) {
	return c.WithTrace(c.http.Post, ctx, url, args...)
}

// TODO: add all the rest of the http methods

func (c *TracingClient) WithTrace(fn HttpFunc, ctx context.Context, url goja.Value, args ...goja.Value) (*http.Response, error) {
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