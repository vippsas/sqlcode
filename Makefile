test: test_mssql

test_mssql:
	docker compose --progress plain -f docker-compose.mssql.yml run test