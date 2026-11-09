# Матрица трассировки

Источник: предоставленная XLSX, лист «ФТНФТ NEW»; полный текст — [requirements.json](sources/requirements.json). ID сохранены. Ни одна строка partial/demo не означает полное выполнение требования продукта.

| ID | Компонент первого среза | Проверка | Статус |
|---|---|---|---|
| REQ-F01 | collector, event_receipt | TestLateSignalAndReplay | partial: SIM events only |
| REQ-F02 | policy.Calculate | TestPolicy | partial: device/SIM/travel only |
| REQ-F03 | policy.Decide | TestPolicy; TestC4HTTPScenarios | demo: assumptions pending approval |
| REQ-F04 | core.Deliver | TestStaleAuthorizationAndDuplicateDelivery | demo: local service transaction |
| REQ-F05 | future slice | team acceptance pending | outside first slice |
| REQ-F06 | core.Confirm | TestConcurrentConfirmation; TestBindingsAndExpiry | partial: demo proof, no crypto |
| REQ-F07 | collector/profile | TestC4HTTPScenarios; TestPolicy | partial: synthetic source only |
| REQ-F08 | policy + test confirmation | TestC4HTTPScenarios | partial: no SMS, independent real path deferred |
| REQ-F09 | future slice | team acceptance pending | outside first slice |
| REQ-F10 | future slice | team acceptance pending | outside first slice |
| REQ-F11 | core/service simulator | TestC4HTTPScenarios | demo: forwarding only |
| REQ-F12 | future slice | team acceptance pending | outside first slice |
| REQ-F13 | future slice | team acceptance pending | outside first slice |
| REQ-F14 | Android confirmation screen | assembleDebug; physical UX review pending | partial |
| REQ-F15 | future slice | team acceptance pending | outside first slice |
| REQ-F16 | future slice | team acceptance pending | outside first slice |
| REQ-F17 | future slice | team acceptance pending | outside first slice |
| REQ-F18 | future slice | team acceptance pending | outside first slice |
| REQ-F19 | future slice | team acceptance pending | outside first slice |
| REQ-F20 | future slice | team acceptance pending | outside first slice |
| REQ-F21 | future slice | team acceptance pending | outside first slice |
| REQ-F22 | future slice | team acceptance pending | outside first slice |
| REQ-F23 | future slice | team acceptance pending | outside first slice |
| REQ-F24 | policy/gateway | TestPolicy; TestScorerUnavailable | partial: local failure modes |
| REQ-F25 | guard lock + service state | TestStaleAuthorizationAndDuplicateDelivery | demo: simulator only |
| REQ-F26 | decision/evidence/audit | TestLateSignalAndReplay; TestAtomicAuditAndImmutableHistory | partial: local journal, no investigation UI |
| REQ-F27 | immutable policy_version | TestAtomicAuditAndImmutableHistory; TestRuntimeRoleACL | partial: no shadow run or activation switch |
| REQ-F28 | future slice | team acceptance pending | outside first slice |
| REQ-F29 | architecture docs | docs/architecture/README.md | documented demo coverage |
| REQ-F30 | collector reevaluation | TestLateSignalAndReplay | partial: no recovery-contact operation |
| REQ-N01 | service simulator | TestC4HTTPScenarios | partial: fixed scenarios only |
| REQ-N02 | future slice | team acceptance pending | outside first slice |
| REQ-N03 | future slice | team acceptance pending | outside first slice |
| REQ-N04 | gateway/risk | scripts/latency.py; docs/load | target pending full load |
| REQ-N05 | profile/freshness | TestPolicy | partial: synthetic freshness |
| REQ-N06 | future slice | team acceptance pending | outside first slice |
| REQ-N07 | future slice | team acceptance pending | outside first slice |
| REQ-N08 | load profile | docs/load/profile.json | pending full simultaneous run |
| REQ-N09 | confirmation/permit/outbox | TestConcurrentConfirmation; TestBindingsAndExpiry; TestExpiredPermit; TestStaleAuthorizationAndDuplicateDelivery | partial: test proof, local execution |
| REQ-N10 | synthetic-only contract | TestC4HTTPScenarios; AGENTS.md | partial: retention cleanup deferred |
| REQ-N11 | runtime ACL | TestRuntimeRoleACL | partial: no real identity/TLS |
| REQ-N12 | Compose/IO timeouts | assembleDebug | partial: no device stability run |
| REQ-N13 | Android result screen | assembleDebug | partial: usability pending |
| REQ-N14 | future slice | team acceptance pending | outside first slice |
| REQ-N15 | future slice | team acceptance pending | outside first slice |
| REQ-N16 | future slice | team acceptance pending | outside first slice |
| REQ-N17 | PostgreSQL + migrations | TestMigrationRoundtrip | pending HA and backup recovery |
| REQ-N18 | own Go rules/service | go test; HTTP demo | demo implemented |
