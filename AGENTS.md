# Repository guide

Учебный стенд с синтетикой. Перед изменениями прочитай git status; сохраняй работу команды и исходные DOCX/XLSX/PNG.

Карта: cmd/{gateway,risk,network-collector}; internal/core — транзакции и симулятор услуги; internal/policy — признаки и тестовая реакция; db — Goose SQL и sqlc запросы; contracts — OpenAPI; apps/mobile — Kotlin Compose; tests — PostgreSQL и HTTP; docs — источники, расхождения, трассировка, результаты. OpenSpec change: protect-call-forwarding.

При изменении поведения читай М-01, docs/discrepancies.md и OpenSpec спецификацию. При изменении хранения/исполнения обязательно читай docs/sources/ADR-007.txt (Accepted для учебного проекта) и design.md изменения. Исходные ID ФТ/НФТ — docs/sources/requirements.json; актуализируй docs/traceability.md.

Проверки: make check; TEST_DATABASE_URL=<isolated-db> make e2e; make generate, затем проверь diff генерируемых файлов; Android инструкции в apps/mobile/AGENTS.md. Windows эквиваленты — scripts/dev.ps1. Go писать прямо, с явными ошибками и транзакционными границами, доступно участнику с опытом Java.

Генерируемые internal/api/*/api.gen.go и internal/store/* меняй через contracts, db/queries, db/migrations и make generate. Версии инструментов закреплены в команде. Повторная генерация должна давать нулевой diff. CI также проверяет появление незакоммиченных generated файлов.

Все изменения защитного состояния, ingest, расход подтверждения и фактическое исполнение сначала блокируют account_guard. Решение+аудит+outbox атомарны; доставка после коммита. Историю решений, политики, аудит и результаты исполнения сохраняй неизменяемыми. Тестовая аттестация и demo-approved годятся только для этого изолированного стенда.

Используй только demo-* аккаунты, sim-* и sink-* назначения. Реальные номера, данные абонентов, токены, ключи, операторские сети исключены. Тесты с БД запускай только на выделенной БД; integration создаёт собственные схемы.

Новый сценарий: .agents/skills/add-fraud-scenario/SKILL.md. Разобранный сбой: .agents/skills/learn-from-failure/SKILL.md. Спорная политика/граница доверия: docs/grill-with-docs.md. CODEOWNERS отсутствует: назначение владельцев открыто; не выдумывай GitHub handles.
