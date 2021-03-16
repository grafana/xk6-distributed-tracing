module github.com/simskij/xk6-distributed-tracing

go 1.15

require (
	github.com/dop251/goja v0.0.0-20210315194146-7e3a2f190116
	github.com/loadimpact/k6 v0.31.0
	github.com/sirupsen/logrus v1.8.1
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.17.0
	go.opentelemetry.io/otel v0.17.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.17.0
	go.opentelemetry.io/otel/sdk v0.17.0
	go.opentelemetry.io/otel/trace v0.17.0
	google.golang.org/grpc v1.35.0 // indirect
)
