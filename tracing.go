package tracing

import (
	"context"
	"math/rand"

	"github.com/k6io/xk6-distributed-tracing/client"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib/consts"
	propb3 "go.opentelemetry.io/contrib/propagators/b3"
	propjaeger "go.opentelemetry.io/contrib/propagators/jaeger"
	propot "go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	exportotlp "go.opentelemetry.io/otel/exporters/otlp"
	exportotlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	exportout "go.opentelemetry.io/otel/exporters/stdout"
	exportjaeger "go.opentelemetry.io/otel/exporters/trace/jaeger"
	exportzipkin "go.opentelemetry.io/otel/exporters/trace/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

const version = "0.0.2"

var attr = resource.NewWithAttributes(
	semconv.ServiceNameKey.String("k6"),
	attribute.String("k6.version", consts.Version),
)

func init() {
	modules.Register(
		"k6/x/tracing",
		&JsModule{
			Version: version,
		})

}

// JsModule exposes the tracing client in the javascript runtime
type JsModule struct {
	Version string
	Http    *client.TracingClient
}

type Options struct {
	Exporter   string
	Propagator string
	Endpoint   string
	CrocoSpans string
}

var initialized bool = false
var provider *tracesdk.TracerProvider
var propagator propagation.TextMapPropagator
var c context.Context

func (*JsModule) XHttp(ctx *context.Context, opts Options) interface{} {
	if !initialized {
		// Set default values
		if opts.Exporter == "" {
			opts.Exporter = "noop"
		}
		if opts.Propagator == "" {
			opts.Propagator = "w3c"
		}

		// Set up propagator
		switch opts.Propagator {
		case "w3c":
			propagator = propagation.TraceContext{}
		case "b3":
			propagator = propb3.B3{}
		case "jaeger":
			propagator = propjaeger.Jaeger{}
		case "ot":
			propagator = propot.OT{}
		default:
			logrus.Error("Unknown tracing propagator")
		}

		// Set up exporter
		switch opts.Exporter {
		case "jaeger":
			if opts.Endpoint == "" {
				opts.Endpoint = "http://localhost:14268/api/traces"
			}
			tp, err := initJaegerProvider(opts.Endpoint)
			if err != nil {
				logrus.WithError(err).Error("Failed to init Jaeger exporter")
			}
			provider = tp
		case "zipkin":
			if opts.Endpoint == "" {
				opts.Endpoint = "http://localhost:9411/api/v2/spans"
			}
			tp, err := initZipkinProvider(opts.Endpoint)
			if err != nil {
				logrus.WithError(err).Error("Failed to init Zipkin exporter")
			}
			provider = tp
		case "otlp":
			if opts.Endpoint == "" {
				opts.Endpoint = "0.0.0.0:55680"
			}
			tp, err := initOtlpProvider(opts.Endpoint)
			if err != nil {
				logrus.WithError(err).Error("Failed to init otlp exporter")
			}
			provider = tp
		case "noop":
			tp, err := initNoopProvider()
			if err != nil {
				logrus.WithError(err).Error("Failed to init Noop exporter")
			}
			provider = tp
		case "stdout":
			tp, err := initStdoutProvider()
			if err != nil {
				logrus.WithError(err).Error("Failed to init stdout exporter")
			}
			provider = tp
		default:
			logrus.Error("Unknown tracing exporter")
		}
		initialized = true
		otel.SetTracerProvider(provider)
		otel.SetTextMapPropagator(propagator)
		// TODO: Use the id generated for the cloud, in case we are running a cloud output test
		testRunID := 100000000000 + rand.Intn(999999999999-100000000000)
		logrus.Info("CrocoSpans test run id: ", testRunID)
		c = context.WithValue(*ctx, "crocospans", client.Vars{Backend: opts.CrocoSpans, TestRunID: testRunID})
	}

	rt := common.GetRuntime(c)
	tracingClient := client.New()
	return common.Bind(rt, tracingClient, &c)
}

func (*JsModule) Shutdown() error {
	return provider.Shutdown(context.Background())
}

func initJaegerProvider(url string) (*tracesdk.TracerProvider, error) {
	exp, err := exportjaeger.NewRawExporter(exportjaeger.WithCollectorEndpoint(exportjaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(attr),
	)
	return tp, nil
}

func initZipkinProvider(url string) (*tracesdk.TracerProvider, error) {
	exp, err := exportzipkin.NewRawExporter(url)
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(attr),
	)
	return tp, nil
}

func initOtlpProvider(url string) (*tracesdk.TracerProvider, error) {
	ctx := context.Background()
	exp, err := exportotlp.NewExporter(ctx, exportotlpgrpc.NewDriver(
		exportotlpgrpc.WithInsecure(),
		exportotlpgrpc.WithEndpoint(url),
	))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(attr),
	)
	return tp, nil
}

func initNoopProvider() (*tracesdk.TracerProvider, error) {
	exportOpts := []exportout.Option{
		exportout.WithoutTraceExport(),
	}
	tp, _, err := exportout.InstallNewPipeline(exportOpts, nil)
	return tp, err
}

func initStdoutProvider() (*tracesdk.TracerProvider, error) {
	exportOpts := []exportout.Option{
		exportout.WithPrettyPrint(),
	}
	tp, _, err := exportout.InstallNewPipeline(exportOpts, nil)
	return tp, err
}
