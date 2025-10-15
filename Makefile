.PHONY: help run build test docker-up docker-down clean

help: ## Mostra esta mensagem de ajuda
	@echo "Comandos disponíveis:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

run: ## Executa a aplicação
	@go run src/cmd/server/main.go

build: ## Compila a aplicação
	@go build -o bin/server src/cmd/server/main.go

docker-up: ## Inicia o GO Feature Flag relay proxy
	@docker-compose up -d

docker-down: ## Para o GO Feature Flag relay proxy
	@docker-compose down

test: ## Executa os testes
	@go test -v ./...

clean: ## Remove arquivos compilados
	@rm -rf bin/
	@go clean

deps: ## Baixa as dependências
	@go mod download
	@go mod tidy

dev: docker-up run ## Inicia o ambiente de desenvolvimento completo

