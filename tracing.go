package tracing


import (
	"context"
	"github.com/loadimpact/k6/js/common"
	"github.com/loadimpact/k6/js/modules"
	"github.com/loadimpact/k6/lib/consts"
	"github.com/simskij/xk6-distributed-tracing/client"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	exportjaeger "go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"os"
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
	Http *client.TracingClient
}

var initialized bool = false

func (*JsModule) XHttp(ctx *context.Context) interface{} {
	if !initialized {
		SetupJaeger()
		initialized = true
	}
	rt := common.GetRuntime(*ctx)
	tracingClient := client.New()
	return common.Bind(rt, tracingClient, ctx)
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