import tracing, { Http } from 'k6/x/tracing';
import { sleep } from 'k6';

export let options = {
  vus: 1,
  duration: '10s',
};

export function setup() {
  console.log(`Running xk6-distributed-tracing v${tracing.version}`);
}

export default function() {
  const http = new Http({
    exporter: "jaeger",
    propagator: "w3c",
    endpoint: "http://localhost:14268/api/traces"
  });
  const r = http.get('https://test-api.k6.io');
  
  console.log(`trace-id=${r.trace_id}`);
  sleep(1);
}

export function teardown(){
  // Cleanly shutdown and flush telemetry when k6 exits.
  tracing.shutdown();
}