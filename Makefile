.PHONY: help up down run test logs clean

help: ## Mostra comandos disponíveis
	@echo "Comandos:"
	@echo "  make up     - Inicia relay proxy + API (Docker)"
	@echo "  make down   - Para todos os containers"
	@echo "  make run    - Roda a API localmente"
	@echo "  make test   - Roda testes"
	@echo "  make logs   - Mostra logs"
	@echo "  make clean  - Remove tudo"

up: ## Inicia relay proxy + API
	@docker-compose up -d
	@echo "Relay proxy: http://localhost:1031"
	@echo "API server:  http://localhost:1324"

down: ## Para Docker
	@docker-compose down

run: ## Roda API localmente
	@export APP_ENV=local && \
	 go run main.go server

test: ## Roda testes
	@go test ./... -race -v

logs: ## Mostra logs
	@docker-compose logs -f

clean: ## Remove tudo
	@docker-compose down -v
	@rm -rf bin/
