
.PHONY: up down build logs ps restart clean test tidy

up:
	docker compose up -d --build

down:
	docker compose down

build:
	docker compose build

logs:
	docker compose logs -f

ps:
	docker compose ps

restart:
	docker compose restart

clean:
	docker compose down -v

test:
	go test ./...

tidy:
	go mod tidy
