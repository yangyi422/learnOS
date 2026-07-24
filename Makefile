.PHONY: dev-api dev-web build-web build-api fmt docker-up docker-prod hash-password

dev-api:
	APP_ENV=development APP_DATA_DIR=./data go run ./cmd/server

dev-web:
	cd web && npm run dev

build-web:
	cd web && npm run build

build-api: build-web
	go build -o ./bin/learnos ./cmd/server

fmt:
	gofmt -w ./cmd ./internal

hash-password:
	@test -n "$(PASSWORD)" || (echo "usage: make hash-password PASSWORD='your-password'" && exit 1)
	@go run ./cmd/hash-password "$(PASSWORD)"

docker-up:
	docker compose up -d --build app

docker-prod:
	docker compose --profile production up -d --build
