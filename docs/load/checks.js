// k6 run -e BASE_URL=http://127.0.0.1:8080 docs/load/checks.js
// Checks-only runner. Use the complete concurrent profile in README.md before claiming ADR acceptance.
import http from 'k6/http';
import { check } from 'k6';
export const options = {
  scenarios: {
    protected_checks: {
      executor: 'ramping-arrival-rate', startRate: 10, timeUnit: '1s', preAllocatedVUs: 40, maxVUs: 200,
      stages: [{target: 10, duration: '1m'}, {target: 10, duration: '30m'}, {target: 100, duration: '1s'}, {target: 100, duration: '10m'}, {target: 10, duration: '1m'}],
    },
  },
  thresholds: { http_req_duration: ['p(99)<=200'], http_req_failed: ['rate==0'], checks: ['rate==1'], dropped_iterations: ['count==0'] },
};
function uuid() { return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => { const r=Math.floor(Math.random()*16); return (c==='x'?r:(r&3|8)).toString(16); }); }
export default function () {
  const response=http.post((__ENV.BASE_URL||'http://127.0.0.1:8080')+'/v1/evaluate',JSON.stringify({accountId:'demo-clean',operationId:uuid(),target:'sink-a',device:'known',attestation:'demo-valid'}),{headers:{'Content-Type':'application/json'}});
  check(response, {'decision saved':r=>r.status===200&&r.json('policyVersion')==='demo-v1'});
}
