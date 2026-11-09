"""Maintain small identical wire schemas for two independently generated Gin APIs."""
import json
from pathlib import Path

def obj(props, required=None):
    return dict(type='object', additionalProperties=False, properties=props, required=required or list(props))
def ref(n): return {'$ref': '#/components/schemas/' + n}
def string(**kw): return dict(type='string', **kw)

schemas = {
    'Evaluation': obj(dict(accountId=string(enum=['demo-fraud','demo-legal','demo-clean']), operationId=string(format='uuid'), target=string(enum=['sink-a','sink-b']), device=string(enum=['known','new']), attestation=string(enum=['demo-valid','unknown']))),
    'Confirmation': obj(dict(accountId=string(enum=['demo-fraud','demo-legal','demo-clean']), operationId=string(format='uuid'), confirmationId=string(format='uuid'), proof=string(enum=['demo-approved']))),
    'Decision': obj(dict(decisionId=string(), result=string(enum=['allow','step_up','hold']), policyVersion=string(), confirmationId=string(), operationId=string()), ['decisionId','result','policyVersion','operationId']),
    'State': obj(dict(target=string(), changes={'type':'integer'}, decisions={'type':'array','items':{'type':'object','additionalProperties':True}}, audit={'type':'array','items':{'type':'object','additionalProperties':True}})),
    'Error': obj(dict(error=string()))
}
for name in ['external','internal']:
    Path('internal/api/'+('external' if name=='external' else 'riskapi')).mkdir(parents=True,exist_ok=True)
    paths = {}
    for route, op, body in [('evaluate','Evaluate','Evaluation'),('confirm','Confirm','Confirmation')]:
        paths['/v1/'+route] = {'post': dict(operationId=op, requestBody=dict(required=True,content={'application/json':{'schema':ref(body)}}),responses={'200':dict(description='decision',content={'application/json':{'schema':ref('Decision')}}),'400':dict(description='invalid request'),'409':dict(description='binding or confirmation rejected'),'503':dict(description='fail closed')})}
    paths['/v1/state/{account}']={'get':dict(operationId='State',parameters=[dict(name='account',**{'in':'path'},required=True,schema=string(enum=['demo-fraud','demo-legal','demo-clean']))],responses={'200':dict(description='synthetic state and audit',content={'application/json':{'schema':ref('State')}}),'503':dict(description='unavailable')})}
    Path('contracts').mkdir(exist_ok=True)
    Path('contracts/'+name+'.json').write_text(json.dumps(dict(openapi='3.0.3', info=dict(title='Synthetic antifraud '+name,version='1.0.0'),paths=paths,components=dict(schemas=schemas)),indent=2)+'\n')
