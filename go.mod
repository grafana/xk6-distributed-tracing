module github.com/k6io/xk6-distributed-tracing

go 1.15

require (
	github.com/dop251/goja v0.0.0-20210317175251-bb14c2267b76
	github.com/loadimpact/k6 v0.31.1
	github.com/sirupsen/logrus v1.8.1
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.18.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/otlp v0.19.0
	go.opentelemetry.io/otel/exporters/stdout v0.19.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.19.0
	go.opentelemetry.io/otel/exporters/trace/zipkin v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
)
