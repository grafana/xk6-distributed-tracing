import { Http } from 'k6/x/tracing';
import { check, group, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '20s', target: 20 },
    { duration: '10s', target: 0 },
  ],
  ext: {
    loadimpact: {
      projectID: 3602528,
      // Test runs with the same name groups test runs together
      name: "Testing HotRod demo"
    }
  }
};

export default function () {
  const http = new Http({"endpoint": "http://localhost:10001/api/v1/push/request"});

  group('do something inside a group', function () {
    let r = http.get('http://localhost:8080/dispatch?customer=123&nonse=0.41418777006108365');
    check(r, {
      'status is 200': (r) => r.status === 200,
    });
    console.log(`traceId=${r.trace_id}`);
    sleep(1);
  });
}