# Сессионный антифрод: protect-call-forwarding

Рабочее учебное демо «Мой МТС», только синтетика. Kotlin/Compose → Gin gateway → Gin risk → PostgreSQL; collector моделирует регистрации SIM, outbox доставляет команду локальному симулятору услуги. Сервер меняет переадресацию только после проверки актуального разрешения в транзакции. Аттестация `demo-valid` и подтверждение `demo-approved` — явно тестовые подстановки, защиты реального аккаунта они не обеспечивают.

Требования: Git, Go 1.25.7, Python 3, Make (либо PowerShell), Docker Engine + Compose v2 с `--wait`. Для Android: JDK 17, SDK 34; Gradle 8.2.1 входит через wrapper. OpenSpec CLI 1.11.0 нужен для работы со спецификацией. Генераторы загружаются закреплёнными версиями; нужен доступ к Go proxy/sumdb, Google Maven и Maven Central. Python demo/checks не требуют pip-пакетов.

Для знакомства с кодом и выбора рабочего процесса: [архитектура проекта и сценарии применения навыков](docs/architecture-and-workflows.md).

Быстрый запуск с Docker:

```sh
make demo
python scripts/demo.py --scenario fraud
python scripts/demo.py --scenario legal
make check
make generate
python scripts/verify_generated.py
TEST_DATABASE_URL='postgres://demo:synthetic-local-only@127.0.0.1:55433/antifraud?sslmode=disable' make e2e
```

`make demo` собирает три процесса, применяет Goose up, запускает сборщик и оба API; скрипт ждёт регистрации сценариев. PostgreSQL доступен только на loopback:55433, gateway на :8080; risk не публикуется на хосте. `docker compose down` останавливает стенд и сохраняет данные. Для повторного чистого прогона создайте отдельный Compose project/volume. После суток fixture-сигналы устаревают: это преднамеренный fail-closed. Очистка существующего volume уничтожает его синтетические результаты, поэтому не включена в demo-команду.

Ожидаемо:

| Сценарий | Решение | Серверная услуга | Аудит |
|---|---|---|---|
| fraud: Калининград → Владивосток, новая SIM/устройство | hold | прежнее состояние, на чистом стенде disabled | версия demo-v1 и снимок причин |
| legal: Калининград → Светлогорск, новая SIM, знакомое устройство | step_up, после тестового подтверждения allow | sink-a, ровно одно изменение на операцию | решение, расход подтверждения в evidence, отдельный факт исполнения |

Политика по М-01 — допущение: высокий риск охлаждает действие на 5мин, средний требует подтверждения; permit/confirmation живут 2мин. Реакции вынесены в `internal/policy`. Фактические правила и пороги команда согласует с ментором. [Расхождения](docs/discrepancies.md) явно сопоставляют C4 и М-01. Неизвестный/устаревший сигнал не равен отрицательному. Повторное включение новой операцией считается отдельным изменением; повтор одной команды не увеличивает счётчик.

Windows без Docker, с установленным PostgreSQL 18:

```powershell
.\scripts\local-demo.ps1
$env:TEST_DATABASE_URL='postgres://demo@127.0.0.1:55432/postgres?sslmode=disable'
.\scripts\dev.ps1 e2e
.\scripts\dev.ps1 check
.\scripts\dev.ps1 generate
.\scripts\dev.ps1 android
```

local-demo создаёт отдельный кластер `.local/pgdata` на loopback:55432 с trust только для локальной синтетики, отдельную `antifraud_demo` и запускает сервисы скрытыми процессами. PID/логи — `.local/`. Мигратор требует права создать роль и таблицы; рабочие соединения переключаются на `antifraud_runtime` с ограниченными grants. Никогда не передавайте URL настоящей БД. Интеграционные тесты создают временные схемы в указанной БД и удаляют только их. Для ручного запуска: `go run ./cmd/risk -migrate up`, `go run ./cmd/network-collector -once`, затем в двух терминалах `go run ./cmd/risk` и `go run ./cmd/gateway` с DATABASE_URL. Сборщик принимает JSON-массив через `-file` (формат core.Registration), сам добавляет received_at.

Android:

```sh
cd apps/mobile
bash gradlew assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

В эмуляторе адрес `http://10.0.2.2:8080` уже задан на экране. Для устройства команды используйте `adb reverse tcp:8080 tcp:8080` и `http://127.0.0.1:8080`. Сначала поднимите backend и collector. Выберите demo-fraud/demo-legal, проверьте действие, для legal нажмите явное тестовое подтверждение, затем «Проверить состояние услуги». HTTP разрешён только для изолированного демо; реальные номера не вводятся. APK собран, физическое устройство и Play Integrity ещё не проверены.

API-контракты: `contracts/external.json`, `contracts/internal.json` (OpenAPI); тела проверяются middleware, HTTP-обвязка и Go-типы сгенерированы oapi-codegen. Изменяйте источник схем в `scripts/contracts.py`; SQL — в db/queries и Goose db/migrations. Генерируемые файлы вручную не редактируются. Шлюз ждёт скорер максимум 1.5с; при ошибке возвращает 503 без изменения услуги.

Документация: [архитектура и приёмочный HA-контур](docs/architecture/README.md), [ADR по стеку](docs/adr/ADR-008-stack.md), [исходные требования и покрытие](docs/traceability.md), [OpenSpec](openspec/changes/protect-call-forwarding/design.md), [фактические проверки](docs/verification.md), [будущий нагрузочный профиль](docs/load/README.md), [grill-with-docs](docs/grill-with-docs.md). ADR-007 сохраняет Accepted для учебного проекта. Автоматическое секционирование/очистка, HA, Play Integrity, выбор passkey/ключа, recovery и отдельный сервис услуги остаются задачами следующего этапа. USSD и кол-центр не подключены.

Проверки CI: Go vet/test, повторная генерация, PostgreSQL integration и migration down/up, HTTP Compose demo, OpenSpec validation, Android assembleDebug. Измерения локального smoke не доказывают p99≤200мс под полной нагрузкой ADR или RPO/RTO. Изменения оставляются для ревью без публикации/деплоя.
