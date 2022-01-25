package tracing

import (
	"context"

	"github.com/dop251/goja"
	"github.com/grafana/xk6-distributed-tracing/client"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/modules"
	k6HTTP "go.k6.io/k6/js/modules/k6/http"
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

const version = "0.1.1"

func init() {
	modules.Register("k6/x/tracing", New())
}

var attr = resource.NewWithAttributes(
	semconv.ServiceNameKey.String("k6"),
	attribute.String("k6.version", consts.Version),
)

type (
	// RootModule is the global module instance that will create DistributedTracing
	// instances for each VU.
	RootModule struct{}

	DistributedTracing struct {
		// modules.VU provides some useful methods for accessing internal k6
		// objects like the global context, VU state and goja runtime.
		vu          modules.VU
		httpRequest client.HttpRequestFunc
	}
)

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Instance = &DistributedTracing{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	r := k6HTTP.New().NewModuleInstance(vu).Exports().Default.(*goja.Object).Get("request")
	var requestFunc client.HttpRequestFunc
	err := vu.Runtime().ExportTo(r, &requestFunc)
	if err != nil {
		panic(err)
	}
	return &DistributedTracing{vu: vu, httpRequest: requestFunc}
}

// Exports implements the modules.Instance interface and returns the exports
// of the JS module.
func (c *DistributedTracing) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"Http":     c.http,
			"shutdown": c.shutdown,
			"version":  version,
		},
	}
}

type Options struct {
	Exporter   string
	Propagator string
	Endpoint   string
}

var (
	initialized bool = false
	provider    *tracesdk.TracerProvider
	propagator  propagation.TextMapPropagator
)

func (t *DistributedTracing) http(call goja.ConstructorCall) *goja.Object {
	rt := t.vu.Runtime()

	obj := call.Argument(0).ToObject(rt)

	opts := Options{
		Exporter:   obj.Get("exporter").ToString().String(),
		Propagator: obj.Get("propagator").ToString().String(),
		Endpoint:   obj.Get("endpoint").ToString().String(),
	}

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
	}

	tracingClient := client.New(t.vu, t.httpRequest)

	return rt.ToValue(tracingClient).ToObject(rt)
}

func (*DistributedTracing) shutdown() error {
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
