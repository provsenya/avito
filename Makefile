generate:
	go generate ./internal/...
	go tool oapi-codegen -generate types,chi-server \
      -include-operation-ids createTrip,getTrip,finishTrip,health,ready \
      -package api -o api/api.gen.go \
      contracts/trip-service.openapi.yaml

migrate:
	@. ./.env; go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	@. ./.env; go tool goose -dir migrations postgres "$$DATABASE_URL" down

migrate-status:
	@. ./.env; go tool goose -dir migrations postgres "$$DATABASE_URL" status

run:
	@set -a; . ./.env.example; . ./.env; set +a; go run ./cmd/trip-service
.PHONY: generate migrate migrate-down migrate-status run test test-integration lint

test:
	go test -race -count=2 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-integration:
	@set -a; . ./.env.example; . ./.env; set +a; go test -race -count=2 -tags=integration ./internal/repository/...

lint:
	go tool golangci-lint run
