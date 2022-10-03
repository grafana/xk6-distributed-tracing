import { Http } from 'k6/x/tracing';
import { check, group, sleep } from 'k6';

export const options = {
  scenarios: {
    smokeDispatch: {
      exec: 'dispatch',
      executor: 'constant-vus',
      vus: 1,
      duration: '15s',
    },
    stressDispatch: {
      exec: 'dispatch',
      executor: 'ramping-vus',
      stages: [
        { duration: '25s', target: 20 },
        { duration: '10s', target: 0 },
      ],
      gracefulRampDown: '5s',
      startTime: '15s',
    },
  },
  ext: {
    loadimpact: {
      projectID: 3602528,
      name: "Testing HotRod demo"
    }
  }
};

const tracingConfig = {"endpoint": "http://localhost:10001/api/v1/push/request"}

const userIDs = ["123", "392", "731", "567"]

export function dispatch() {
  const http = new Http(tracingConfig);
  group('dispatch customer 123', function () {
    let r = http.get(`http://localhost:8080/dispatch?customer=${userIDs[0]}`);
    check(r, {
      'status is 200': (r) => r.status === 200,
    });
    console.log(`trace_id=${r.trace_id} status=${r.status} method=${r.request.method} duration=${r.timings.duration}`)
  });
  sleep(1);
}