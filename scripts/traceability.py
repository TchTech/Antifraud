"""Rebuild coverage index using original requirement IDs without changing their text."""
import json
from pathlib import Path

coverage={
 'REQ-F01':('collector, event_receipt','TestLateSignalAndReplay','partial: SIM events only'),
 'REQ-F02':('policy.Calculate','TestPolicy','partial: device/SIM/travel only'),
 'REQ-F03':('policy.Decide','TestPolicy; TestC4HTTPScenarios','demo: assumptions pending approval'),
 'REQ-F04':('core.Deliver','TestStaleAuthorizationAndDuplicateDelivery','demo: local service transaction'),
 'REQ-F06':('core.Confirm','TestConcurrentConfirmation; TestBindingsAndExpiry','partial: demo proof, no crypto'),
 'REQ-F07':('collector/profile','TestC4HTTPScenarios; TestPolicy','partial: synthetic source only'),
 'REQ-F08':('policy + test confirmation','TestC4HTTPScenarios','partial: no SMS, independent real path deferred'),
 'REQ-F11':('core/service simulator','TestC4HTTPScenarios','demo: forwarding only'),
 'REQ-F14':('Android confirmation screen','assembleDebug; physical UX review pending','partial'),
 'REQ-F24':('policy/gateway','TestPolicy; TestScorerUnavailable','partial: local failure modes'),
 'REQ-F25':('guard lock + service state','TestStaleAuthorizationAndDuplicateDelivery','demo: simulator only'),
 'REQ-F26':('decision/evidence/audit','TestLateSignalAndReplay; TestAtomicAuditAndImmutableHistory','partial: local journal, no investigation UI'),
 'REQ-F27':('immutable policy_version','TestAtomicAuditAndImmutableHistory; TestRuntimeRoleACL','partial: no shadow run or activation switch'),
 'REQ-F29':('architecture docs','docs/architecture/README.md','documented demo coverage'),
 'REQ-F30':('collector reevaluation','TestLateSignalAndReplay','partial: no recovery-contact operation'),
 'REQ-N01':('service simulator','TestC4HTTPScenarios','partial: fixed scenarios only'),
 'REQ-N04':('gateway/risk','scripts/latency.py; docs/load','target pending full load'),
 'REQ-N05':('profile/freshness','TestPolicy','partial: synthetic freshness'),
 'REQ-N08':('load profile','docs/load/profile.json','pending full simultaneous run'),
 'REQ-N09':('confirmation/permit/outbox','TestConcurrentConfirmation; TestBindingsAndExpiry; TestExpiredPermit; TestStaleAuthorizationAndDuplicateDelivery','partial: test proof, local execution'),
 'REQ-N10':('synthetic-only contract','TestC4HTTPScenarios; AGENTS.md','partial: retention cleanup deferred'),
 'REQ-N11':('runtime ACL','TestRuntimeRoleACL','partial: no real identity/TLS'),
 'REQ-N12':('Compose/IO timeouts','assembleDebug','partial: no device stability run'),
 'REQ-N13':('Android result screen','assembleDebug','partial: usability pending'),
 'REQ-N17':('PostgreSQL + migrations','TestMigrationRoundtrip','pending HA and backup recovery'),
 'REQ-N18':('own Go rules/service','go test; HTTP demo','demo implemented'),
}
rows=json.loads(Path('docs/sources/requirements.json').read_text(encoding='utf8'))
lines=['# Матрица трассировки','','Источник: предоставленная XLSX, лист «ФТНФТ NEW»; полный текст — [requirements.json](sources/requirements.json). ID сохранены. Ни одна строка partial/demo не означает полное выполнение требования продукта.','','| ID | Компонент первого среза | Проверка | Статус |','|---|---|---|---|']
for r in rows:
    component,check,status=coverage.get(r['id'],('future slice','team acceptance pending','outside first slice'))
    lines.append('| '+' | '.join([r['id'],component,check,status])+' |')
Path('docs/traceability.md').write_text('\n'.join(lines)+'\n',encoding='utf8')
