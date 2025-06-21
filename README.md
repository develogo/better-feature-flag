# Better Feature Flag

Este é um projeto Go integrado com GO Feature Flag para demonstrar como implementar feature flags em uma aplicação.

## Estrutura do Projeto

- `main.go` - API Go principal com integração do GO Feature Flag
- `flags.goff.yaml` - Configuração das feature flags
- `goff-proxy.yaml` - Configuração do relay proxy
- `docker-compose.yml` - Para executar o relay proxy facilmente

## Como Executar

### 1. Instalar Dependências
```bash
go mod tidy
```

### 2. Iniciar o GO Feature Flag Relay Proxy

**Opção 1 - Usando Docker Compose (mais fácil):**
```bash
docker-compose up -d
```

**Opção 2 - Comando Docker direto (como na documentação oficial):**
```bash
docker run -p 1031:1031 -v $(pwd)/flags.goff.yaml:/goff/flags.goff.yaml -v $(pwd)/goff-proxy.yaml:/goff/goff-proxy.yaml gofeatureflag/go-feature-flag:latest
```

### 3. Executar a API
```bash
go run main.go
```

### 4. Testar a API

Sem mostrar email (usuário padrão):
```bash
curl -H "X-USER-ID: 2" http://localhost:1323
```

Com email (usuário com ID 1):
```bash
curl -H "X-USER-ID: 1" http://localhost:1323
```

## Configuração das Feature Flags

As flags estão configuradas no arquivo `flags.goff.yaml`. Você pode modificar as regras de targeting e ver as mudanças em tempo real sem reiniciar a aplicação.

### Exemplo de configuração avançada:

```yaml
show-email-contact:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: targetingKey eq "1"
      variation: enabled
  defaultRule:
    variation: disabled
```

## Endpoints

- `GET /` - Retorna informações do usuário, incluindo email se a flag estiver habilitada para o usuário 