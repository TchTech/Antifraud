"""Sequential HTTP smoke histogram. Not the ADR simultaneous load acceptance."""
import argparse,json,math,platform,time,urllib.request,uuid
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--url',default='http://127.0.0.1:8080');p.add_argument('--count',type=int,default=200);args=p.parse_args()
latencies=[];started=time.perf_counter()
for i in range(args.count+10):
    b=dict(accountId='demo-clean',operationId=str(uuid.uuid4()),target='sink-a',device='known',attestation='demo-valid')
    req=urllib.request.Request(args.url+'/v1/evaluate',data=json.dumps(b).encode(),headers={'Content-Type':'application/json'})
    start=time.perf_counter()
    with urllib.request.urlopen(req,timeout=5) as r:
        d=json.load(r)
        assert d['result']=='allow',d
    if i>=10: latencies.append((time.perf_counter()-start)*1000)
latencies.sort()
out=dict(kind='sequential smoke, not ADR load acceptance',count=len(latencies),warmup=10,concurrency=1,platform=platform.platform(),python=platform.python_version(),p50Ms=latencies[math.ceil(.5*len(latencies))-1],p95Ms=latencies[math.ceil(.95*len(latencies))-1],p99Ms=latencies[math.ceil(.99*len(latencies))-1],maxMs=max(latencies),elapsedSeconds=time.perf_counter()-started)
Path('docs/load/local-smoke.json').write_text(json.dumps(out,indent=2)+'\n')
print(json.dumps(out,indent=2))
