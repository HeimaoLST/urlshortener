postgres:
	docker run --name postgres17 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -p 5432:5432 -d postgres
createdb:
	docker exec -it postgres17 createdb --username=root --owner=root urlshortener
dropdb:
	docker exec -it postgres17 dropdb --username=root urlshortener
migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/urlshortener?sslmode=disable" -verbose up
migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/urlshortener?sslmode=disable" -verbose down
sqlc:
	docker run --rm -v "${CURDIR}:/src" -w /src sqlc/sqlc generate
server:
	go build main.go && .\main.exe
	
.PHONY: createdb dropdb migrateup migratedown sqlc server