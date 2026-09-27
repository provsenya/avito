generate:
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