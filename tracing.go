package tracing

import (
	"context"
	"os"

	"github.com/k6io/xk6-distributed-tracing/client"
	"github.com/loadimpact/k6/js/common"
	"github.com/loadimpact/k6/js/modules"
	"github.com/loadimpact/k6/lib/consts"
	"github.com/sirupsen/logrus"
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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

const version = "0.0.1"

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
}

var initialized bool = false

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
		var propagator propagation.TextMapPropagator
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
			initJaegerTracer()
			otel.SetTextMapPropagator(propagator)
		case "zipkin":
			initZipkinTracer()
			otel.SetTextMapPropagator(propagator)
		case "noop":
			initNoopTracer()
			otel.SetTextMapPropagator(propagator)
		case "stdout":
			initStdoutTracer()
			otel.SetTextMapPropagator(propagator)
		case "otlp":
			initOtlpTracer()
			otel.SetTextMapPropagator(propagator)
		default:
			logrus.Error("Unknown tracing exporter")
		}
		initialized = true
	}
	rt := common.GetRuntime(*ctx)
	tracingClient := client.New()
	return common.Bind(rt, tracingClient, ctx)
}

func initJaegerTracer() {
	jaegerEndpoint, ok := os.LookupEnv("JAEGER_ENDPOINT")
	if !ok {
		jaegerEndpoint = "http://localhost:14268/api/traces"
	}
	_, err := exportjaeger.InstallNewPipeline(
		exportjaeger.WithCollectorEndpoint(jaegerEndpoint),
		exportjaeger.WithSDKOptions(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.ServiceNameKey.String("k6"),
				attribute.String("exporter", "zipkin"),
				attribute.String("k6.version", consts.Version),
			)),
		),
	)

	if err != nil {
		logrus.WithError(err).Error("Error while starting the Jaeger exporter pipeline")
	} else {
		logrus.Info("Jaeger exporter configured")
	}
}

func initZipkinTracer() {
	// Create a Zipkin exporter and install it as a global tracer.
	zipkinEndpoint, ok := os.LookupEnv("ZIPKIN_ENDPOINT")
	if !ok {
		zipkinEndpoint = "http://localhost:9411/api/v2/spans"
	}
	err := exportzipkin.InstallNewPipeline(
		zipkinEndpoint,
		exportzipkin.WithSDKOptions(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.ServiceNameKey.String("k6"),
				attribute.String("exporter", "zipkin"),
				attribute.String("k6.version", consts.Version),
			)),
		),
	)
	if err != nil {
		logrus.WithError(err).Error("Error while starting the Zipkin exporter pipeline")
	}
}

func initNoopTracer() {
	// Create a noop exporter and install it as a global tracer.
	exportOpts := []exportout.Option{
		exportout.WithoutTraceExport(),
	}
	_, err := exportout.InstallNewPipeline(exportOpts, nil)
	if err != nil {
		logrus.WithError(err).Error("Error while starting the noop exporter pipeline")
	}
}

func initStdoutTracer() {
	// Create a stdout exporter and install it as a global tracer.
	exportOpts := []exportout.Option{
		exportout.WithPrettyPrint(),
	}
	_, err := exportout.InstallNewPipeline(exportOpts, nil)
	if err != nil {
		logrus.WithError(err).Error("Error while starting the stdout exporter pipeline")
	}
}

func initOtlpTracer() {
	// Create an otlp exporter and install it as a global tracer.
	// TODO: Replace this with otlp.InstallNewPipeline() https://github.com/open-telemetry/opentelemetry-go/pull/1373
	ctx := context.Background()

	otelAgentAddr, ok := os.LookupEnv("OTEL_AGENT_ENDPOINT")
	if !ok {
		otelAgentAddr = "0.0.0.0:55680"
	}

	exp, err := exportotlp.NewExporter(ctx, exportotlpgrpc.NewDriver(
		exportotlpgrpc.WithInsecure(),
		exportotlpgrpc.WithEndpoint(otelAgentAddr),
	))
	if err != nil {
		logrus.WithError(err).Error("Failed to create otlp exporter")
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("k6"),
		),
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to create otlp resource")
	}

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tracerProvider)
}
