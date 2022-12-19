write-mysql-up:
	docker run --name write-mysql -p 33061:3306 -e MYSQL_ROOT_PASSWORD=1234 -d mysql:8.0.28

read-mysql-up:
	docker run --name read-mysql -p 33062:3306 -e MYSQL_ROOT_PASSWORD=1234 -d mysql:8.0.28

mysql-up:
	write-mysql-up
	read-mysql-up

MIGRATION_PATH="db/migrations"
WRITE_DB_URL="mysql://root:1234@(127.0.0.1:33061)/go_example?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=UTC"
READ_DB_URL="mysql://root:1234@(127.0.0.1:33062)/go_example?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=UTC"

## migrate up or down
db-migrate-up:
	migrate -database ${WRITE_DB_URL} -path db/migrations -verbose up
	migrate -database ${READ_DB_URL} -path db/migrations -verbose up
db-migrate-down:
	migrate -database ${WRITE_DB_URL} -path db/migrations -verbose down
	migrate -database ${READ_DB_URL} -path db/migrations -verbose down

## create or alter table command
migrate-create:
	migrate create -ext sql -dir migrations init_users_table