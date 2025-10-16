.PHONY: help up down run logs clean

help: ## Mostra comandos disponíveis
	@echo "Comandos:"
	@echo "  make up     - Inicia relay proxy (Docker)"
	@echo "  make down   - Para relay proxy"
	@echo "  make run    - Roda a API localmente"
	@echo "  make logs   - Mostra logs"
	@echo "  make clean  - Remove tudo"

up: ## Inicia relay proxy
	@docker-compose up -d
	@echo "✅ Relay proxy rodando em http://localhost:1031"

down: ## Para Docker
	@docker-compose down

run: ## Roda API localmente
	@export APP_ENV=local && \
	 go run main.go server

logs: ## Mostra logs
	@docker-compose logs -f

clean: ## Remove tudo
	@docker-compose down -v
	@rm -rf bin/
