ARGS=$(filter-out $@,$(MAKECMDGOALS))

.PHONY: producer
producer: 
	docker compose exec kafka-local kafka-console-producer.sh --bootstrap-server kafka-local:9092 --topic topic

migrate-up:
	migrate -path ./migrations -database 'postgres://postgres:1@0.0.0.0:5432/postgres?sslmode=disable' up
migrate-down:
	migrate -path ./migrations -database 'postgres://postgres:1@0.0.0.0:5432/postgres?sslmode=disable' down
topics:
	docker exec -it kafka-local kafka-topics.sh --bootstrap-server kafka-local:9092 --list
messages:
	docker exec -it kafka-local kafka-console-consumer.sh --bootstrap-server kafka-local:9092 --topic topic --from-beginning
topic:
	kafka-topics.sh --create --topic topic --bootstrap-server kafka-local:9092
testAll:
	go test -v -cover -coverpkg ./... ./...
dockerRun:
	docker build -t app . && docker compose up -d
dockerClear:
	docker compose down -v && docker rmi app
test-e2e: test-db-up
	@echo "Waiting for test database to be ready..."
	@sleep 3
	@echo "Running E2E tests..."
	TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/test_db?sslmode=disable" \
		go test -v -race -count=1 -tags=e2e ./test/e2e/...
