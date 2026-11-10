# Фактическая проверка 2026-11-09

Изменения оставлены в рабочем дереве для ревью, без commit/push. Приложенные исходники DOCX/XLSX/PNG сохранены. Во время работы появились сторонние навыки команды, .claude/.zcode и skills-lock.json: они сохранены, в этот срез их установка не входит.

## Локальный стенд

Windows 11 build 26200, AMD Ryzen 5 7640HS (6 cores / 12 logical), RAM 16 312 553 472 байт. Go 1.25.7 windows/amd64, Python 3.14.3. Отдельный PostgreSQL **18.6** из установленного окружения, кластер `.local/pgdata`, loopback:55432, отдельная БД antifraud_demo. shared_buffers=128MB, max_connections=100, fsync=on, synchronous_commit=on, синхронных реплик нет. pgxpool по умолчанию; три синтетических аккаунта. Настройка диска не измерена; это не приёмочный стенд. Compose/CI закрепляют PostgreSQL 18.1.

## Что запускалось

| Проверка | Фактический результат |
|---|---|
| Go dependency checks | go mod tidy завершён; sumdb не отключалась, из-за DNS использовано зеркало goproxy.cn |
| Go static/unit | go vet ./... и go test ./... прошли; unit-контроли low/medium/high, unknown, stale, одиночная география |
| Генерация | oapi-codegen 2.4.1 + sqlc 1.29.0; повторная генерация 7 файлов даёт те же SHA256 |
| PostgreSQL / HTTP | 10 integration-тестов, в том числе подслучаи duplicate/stale, прошли на настоящем PostgreSQL |
| Миграции | Goose up, полный down до 0 и повторный up на отдельных тестовых схемах; обновление demo-БД без удаления истории |
| Android | Gradle 8.2.1 / AGP 8.1.1 / Kotlin 1.9.10 / Compose compiler 1.5.3, JDK 21.0.8 локально, SDK 34: assembleDebug прошёл; CI задаёт JDK 17 |
| OpenSpec | init --tools codex (CLI 1.11.0), validate protect-call-forwarding --strict — успешно |
| Навыки | quick_validate.py — add-fraud-scenario и learn-from-failure валидны |
| Compose | docker compose config --quiet — успешно; запуск контейнеров не проверен: Docker daemon/pipe отсутствует |
| Отдельные процессы | scripts/local-demo.ps1: сборка трёх бинарников, Goose, collector, скрытые gateway/risk; оба scripts/demo.py сценария прошли, повторный запуск также |

В integration действительно проверяются: C4 через внешний HTTP gateway и внутренний Gin API с проверкой **услуги**, решения и аудита; отклонение неизвестных полей и неверного аккаунта; stale revision после запрета; expiry; дедупликация доставки; конкурентное расходование одного подтверждения (1 успех / 1 конфликт); подмена параметров/аккаунта/операции; поздняя регистрация и ссылка supersedes при неизменном старом evidence; replay с другим hash; полный откат решения/outbox при недоступном аудите; запрет UPDATE/DELETE/TRUNCATE/отключения триггеров runtime-ролью; 503 при недоступном скорере.

Первый чистый HTTP-прогон отдельных процессов:

```json
{"scenario":"fraud","decision":"hold","target":"disabled","changes":0,"audit":3}
{"scenario":"legal","decision":"step_up","target":"sink-a","changes":1,"audit":5}
```

Следующие legal-прогоны используют новый operation_id и увеличивают changes на один каждый. Повтор той же команды отдельно проверен интеграционным тестом и не увеличивает changes.

## Измерение задержки

`python scripts/latency.py`: 10 разогревающих + 200 последовательных запросов, concurrency=1, полная HTTP проверка до commit решения. Сырые результаты и платформа в [local-smoke.json](load/local-smoke.json). Последний замер после миграции 2: p50 2.99мс, p95 27.33мс, p99 29.35мс, max 30.36мс, 200/200 успешны, длительность 1.88с. Outbox доставляется фоном отдельно. В предыдущем smoke наблюдался max 251.68мс; повторный результат не отменяет этот наблюдавшийся выброс.

Это smoke на маленькой истории, **не доказательство** p99≤200мс под совместными потоками ADR-007. Максимум одного запроса также не равен p99. [Будущий профиль](load/README.md) задаёт 100/1000 событий/с, 10/100 проверок/с и предварительные 1000 регистраций/с, всплеск 10мин. Полный генератор обычных session events, prefill истории и расширение аккаунтов нуждаются в следующем срезе; executable k6 checks.js проверяет только HTTP-поток.

## Не проверено / решения команды

GitHub Actions настроены, удалённый workflow из этой сессии не запускался. Docker executable есть, daemon недоступен; доступная замена — установленный изолированный PostgreSQL, а не in-memory mock. HA P/S1/S2/A, fencing, RPO=0/RTO≤5мин, backup/WAL restore, аналитические SLA и full volume/load не проверялись. Их нельзя вывести из локального демо.

APK собран, но установка/визуальная приёмка на физическом устройстве, реальные Play Integrity, passkey/device key, recovery, TLS/identity и ресурсные бюджеты Android не выполнялись. Тестовый HTTP и demo-approved предназначены только для изолированной синтетики.

Команда/ментор согласуют фактическую политику (оба расхождения C4/М-01), пороги и TTL, официальный источник SIM-swap, независимое подтверждение и recovery, протокол отдельного владельца услуги, production роли/partition/lifecycle, HA и владельцев PR. Открытые задачи сохранены в OpenSpec; исходный ADR-007 остаётся Accepted для учебного проекта.
