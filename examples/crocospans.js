import tracing, { Http } from 'k6/x/tracing';
import { group, sleep } from 'k6';

export let options = {
  vus: 1,
  duration: '10s',
};

export function setup() {
  console.log(`Running xk6-distributed-tracing v${tracing.version}`);
}

export default function () {
  const http = new Http({
    "exporter": "otlp",
    "propagator": "w3c",
    "croco_spans": "http://localhost:8081/ingest"
  });
  group('do something inside a group', function () {
    http.get('http://localhost:5000/foo');
    sleep(1);
  });
  http.get('http://localhost:5000/foo');
}

export function teardown() {
  // Cleanly shutdown and flush telemetry when k6 exits.
  tracing.shutdown();
}