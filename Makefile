test: test-mssql

.PHONY:
test-mssql:
	docker compose --progress plain -f docker-compose.mssql.yml run test