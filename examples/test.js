import tracing, { Http } from 'k6/x/tracing';
import { sleep } from 'k6';

export function setup() {
  console.log(`Running xk6-distributed-tracing v${tracing.version}`, tracing)
}

export default function() {
  const http = new Http();
  const r = http.get('https://test-api.k6.io');
  
  console.log(JSON.stringify(r.request.headers))
  sleep(1)
}