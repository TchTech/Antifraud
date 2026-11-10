"""Exercise actual HTTP gateway and assert server side state, decisions and audit."""
import argparse,json,time,uuid,urllib.request,urllib.error

parser=argparse.ArgumentParser()
parser.add_argument('--scenario',choices=['fraud','legal','both'],default='both')
parser.add_argument('--url',default='http://127.0.0.1:8080')
args=parser.parse_args()

def call(path,body=None):
    req=urllib.request.Request(args.url+path,data=json.dumps(body).encode() if body else None,headers={'Content-Type':'application/json'})
    with urllib.request.urlopen(req,timeout=5) as response: return json.load(response)

for name in (['fraud','legal'] if args.scenario=='both' else [args.scenario]):
    account='demo-'+name
    for attempt in range(60):
        try:
            before=call('/v1/state/'+account)
            if len(before['audit'])>=2: break
        except (OSError,urllib.error.URLError): pass
        time.sleep(.5)
    else: raise RuntimeError('stand is not ready or collector did not seed fixtures')
    op=str(uuid.uuid4())
    decision=call('/v1/evaluate',dict(accountId=account,operationId=op,target='sink-a',device='new' if name=='fraud' else 'known',attestation='demo-valid'))
    assert decision['result']==('hold' if name=='fraud' else 'step_up'),decision
    state=call('/v1/state/'+account)
    assert state['changes']==before['changes'],state
    if name=='legal':
        allowed=call('/v1/confirm',dict(accountId=account,operationId=op,confirmationId=decision['confirmationId'],proof='demo-approved'))
        assert allowed['result']=='allow',allowed
        for attempt in range(60):
            state=call('/v1/state/'+account)
            if state['changes']==before['changes']+1: break
            time.sleep(.1)
        assert state['target']=='sink-a' and state['changes']==before['changes']+1,state
        assert any(a['action']=='execution' and a['details']['result']=='executed' for a in state['audit'])
    else: assert state['target']==before['target']
    assert any(d['operation_id']==op and d['policy_version']=='demo-v1' and d['evidence'] for d in state['decisions'])
    assert any(a['operation_id']==op and a['action']=='decision' for a in state['audit'])
    print(json.dumps(dict(scenario=name,decision=decision['result'],target=state['target'],changes=state['changes'],audit=len(state['audit'])),ensure_ascii=False))
