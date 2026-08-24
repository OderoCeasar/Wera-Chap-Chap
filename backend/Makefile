DB_URL ?= postgres://wera:wera@localhost:5432/wera_chap_chap?sslmode=disable

network:
	docker network create wera-network

postgres:
	docker run --name wera-postgres --network wera-network -p 5432:5432 \
		-e POSTGRES_USER=wera -e POSTGRES_PASSWORD=wera -d postgres:16-alpine

createdb:
	docker exec -it wera-postgres createdb --username=wera --owner=wera wera_chap_chap

dropdb:
	docker exec -it wera-postgres dropdb --username=wera wera_chap_chap

# The server runs migrations itself on startup; these targets are for driving
# the chain by hand -- checking a down migration works, or stepping back one.
migrateup:
	migrate -path db/migrations -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migrations -database "$(DB_URL)" -verbose down

migrateup1:
	migrate -path db/migrations -database "$(DB_URL)" -verbose up 1

migratedown1:
	migrate -path db/migrations -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migrations -seq $(name)

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

run:
	go run main.go

build:
	go build -o bin/wera-api .

.PHONY: network postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration sqlc test run build
