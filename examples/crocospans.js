import { Http } from 'k6/x/tracing';
import { check, group, sleep } from 'k6';

export let options = {
  vus: 1,
  duration: '1000s',
};

export default function () {
  const http = new Http({"endpoint": "http://localhost:10001/api/v1/push/request"});

  group('do something inside a group', function () {
    let r = http.get('http://localhost:8080/dispatch?customer=123&nonse=0.41418777006108365');
    check(r, {
      'status is 200': (r) => r.status === 200,
    });
    console.log(`traceId=${r.trace_id}`);
    console.log(r.body)
    sleep(1);
  });
}