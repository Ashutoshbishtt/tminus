.DEFAULT_GOAL := help
.PHONY: help up down ps logs reset tools tools-down psql redis kafka-topics rabbit-ui

help: ## show this list
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## start the local stack and wait until everything is healthy
	docker compose up -d --wait
	@echo
	@$(MAKE) --no-print-directory ps

down: ## stop the stack, keep the data
	docker compose down

ps: ## what is running and how it is doing
	@docker compose ps --format 'table {{.Service}}\t{{.Status}}\t{{.Ports}}'

logs: ## follow the logs (make logs s=kafka for just one)
	docker compose logs -f $(s)

reset: ## stop everything and throw the data away
	docker compose down -v

tools: ## start the management UIs as well (Kafka UI on :8080)
	docker compose --profile tools up -d --wait

tools-down: ## stop just the management UIs
	docker compose --profile tools stop kafka-ui

psql: ## open a psql shell
	docker compose exec postgres psql -U tminus -d tminus

redis: ## open a redis shell
	docker compose exec redis redis-cli

kafka-topics: ## list the kafka topics
	docker compose exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

rabbit-ui: ## open the RabbitMQ management UI
	open http://localhost:15672
