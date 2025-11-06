test:
	docker compose --progress plain -f docker-compose.mssql.yml run test

test_pgsql:
	docker compose --progress plain -f docker-compose.pgsql.yml run test