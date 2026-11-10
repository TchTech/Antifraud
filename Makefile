.PHONY: generate check demo migrate e2e android
generate:
	python scripts/contracts.py
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config contracts/external-codegen.yaml contracts/external.json
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config contracts/internal-codegen.yaml contracts/internal.json
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate
check:
	go vet ./...
	go test ./...
migrate:
	go run ./cmd/risk -migrate up
e2e:
	go test -tags integration -count=1 -v ./tests/...
demo:
	docker compose up --build -d --wait
	python scripts/demo.py
android:
	cd apps/mobile && bash gradlew assembleDebug
