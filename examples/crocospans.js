import { Http } from 'k6/x/tracing';
import { check, group, sleep } from 'k6';

export let options = {
  vus: 1,
  duration: '10s',
};

export default function () {
  const http = new Http({"endpoint": "http://localhost:8081/inges"});

  group('do something inside a group', function () {
    let r = http.get('http://localhost:5000/foo');
    check(r, {
      'status is 200': (r) => r.status === 200,
    });
    console.log(`trace-id=${r.trace_id}`);
    sleep(1);
  });

  let r = http.get('http://localhost:5000/foo');
  check(r, {
    'status is 200': (r) => r.status === 200,
  });
  console.log(`trace-id=${r.trace_id}`);
  console.log(JSON.stringify(r.request.headers))
}