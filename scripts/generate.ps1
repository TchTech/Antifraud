$ErrorActionPreference = 'Stop'
python scripts/contracts.py
if ($LASTEXITCODE) { exit $LASTEXITCODE }
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config contracts/external-codegen.yaml contracts/external.json
if ($LASTEXITCODE) { exit $LASTEXITCODE }
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config contracts/internal-codegen.yaml contracts/internal.json
if ($LASTEXITCODE) { exit $LASTEXITCODE }
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate
exit $LASTEXITCODE
