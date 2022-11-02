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
  const http = new Http();
  const r = http.get('https://test-api.k6.io');
  
  console.log(`trace-id=${r.trace_id}`);
  sleep(1);
}