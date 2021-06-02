module github.com/k6io/xk6-distributed-tracing

go 1.15

require (
	github.com/dop251/goja v0.0.0-20210427212725-462d53687b0d
	github.com/sirupsen/logrus v1.8.1
	go.k6.io/k6 v0.32.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.20.0
	go.opentelemetry.io/contrib/propagators v0.20.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/otlp v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.20.0
	go.opentelemetry.io/otel/exporters/trace/zipkin v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
)
