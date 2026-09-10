.PHONY: up down logs build test load-test migrate-up migrate-down seed psql clean

## Start Postgres, Redis, Jaeger, run migrations, start the API, Prometheus,
## Grafana, and the Streamlit demo UI (all in Docker)
up:
	docker compose up --build -d
	@echo "Waiting for the API to become healthy..."
	@sleep 3
	@curl -sf http://localhost:8080/status/live > /dev/null && echo "\nSeki API is up.\n" || echo "\nAPI not ready yet, check: docker compose logs -f api\n"
	@echo "  API:         http://localhost:8080/api/v1"
	@echo "  Demo UI:     http://localhost:8501"
	@echo "  Grafana:     http://localhost:3000  (admin/admin)"
	@echo "  Prometheus:  http://localhost:9090"
	@echo "  Jaeger:      http://localhost:16686"

## Stop and remove all containers (keeps the Postgres volume)
down:
	docker compose down

## Stop everything AND wipe the database volume (clean slate)
clean:
	docker compose down -v

## Tail logs from every service
logs:
	docker compose logs -f

## Rebuild the API image only
build:
	docker compose build api

## Run Go unit + integration tests (requires local Go toolchain)
test:
	go test ./... -v -race -cover

## Run the k6 load test proving no double-bookings under concurrency
## (requires k6: https://k6.io/docs/get-started/installation/)
load-test:
	k6 run test/load/booking_load_test.js

## Apply migrations manually against a running Postgres (useful outside Docker)
migrate-up:
	migrate -path migrations -database "$${DATABASE_URL:-postgres://seki:seki@localhost:5432/seki?sslmode=disable}" up

## Roll back the last migration
migrate-down:
	migrate -path migrations -database "$${DATABASE_URL:-postgres://seki:seki@localhost:5432/seki?sslmode=disable}" down 1

## Seed demo users + a demo resource
seed:
	docker exec -i seki-postgres psql -U seki -d seki < scripts/seed.sql

## Open a psql shell against the running Postgres container
psql:
	docker exec -it seki-postgres psql -U seki -d seki
