.PHONY: up down logs up-test-login down-test-login logs-test-login

up:
	docker compose -f deploy/compose/default/docker-compose.yml up -d --build

down:
	docker compose -f deploy/compose/default/docker-compose.yml down

logs:
	docker compose -f deploy/compose/default/docker-compose.yml logs -f

up-test-login:
	docker compose -f deploy/compose/testpage/docker-compose.yml up -d --build

down-test-login:
	docker compose -f deploy/compose/testpage/docker-compose.yml down

logs-test-login:
	docker compose -f deploy/compose/testpage/docker-compose.yml logs -f
