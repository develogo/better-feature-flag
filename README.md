# Better Feature Flag

Serviço Go para gerenciamento de feature flags integrado com GO Feature Flag, otimizado para consumo por aplicações Flutter.

## Arquitetura

O projeto segue a arquitetura limpa (Clean Architecture) com separação em camadas:

```
src/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── config/                 # Configurações e variáveis de ambiente
│   ├── handlers/               # HTTP handlers (flags, health)
│   ├── models/                 # Structs de request/response
│   ├── middleware/             # JWT, logging, CORS
│   └── services/               # Lógica de negócio (feature flags)
```

## Features

- ✅ Feature flags com GO Feature Flag
- ✅ Autenticação JWT opcional
- ✅ Logs estruturados em JSON
- ✅ Health checks (liveness e readiness)
- ✅ CORS configurado para Flutter
- ✅ Graceful shutdown
- ✅ Configuração via variáveis de ambiente

## Requisitos

- Go 1.23+
- Docker (para GO Feature Flag relay proxy)

## Configuração

1. Copie o arquivo de exemplo:
```bash
cp .env.example .env
```

2. Edite `.env` com suas configurações:
```env
JWT_SECRET=sua-chave-secreta-aqui
SERVER_PORT=1324
GOFF_ENDPOINT=http://localhost:1031
ENVIRONMENT=development
```

⚠️ **Importante**: Em produção, use variáveis de ambiente seguras e não commite o arquivo `.env`.

## Ambientes

O projeto suporta 3 ambientes com flags isoladas:

- **Development** - Para desenvolvimento local (`flags/*/dev.yaml`)
- **Staging** - Para testes e homologação (`flags/*/staging.yaml`)
- **Production** - Para produção (`flags/*/production.yaml`)

## Como Executar

### Opção 1: Usando Makefile (Recomendado)

```bash
# Development
make dev-up      # Inicia ambiente de desenvolvimento
make dev-logs    # Mostra logs
make dev-down    # Para ambiente

# Staging
make staging-up

# Production (cuidado!)
make prod-up
```

### Opção 2: Manual

#### 1. Instalar Dependências
```bash
go mod download
```

#### 2. Configurar variáveis de ambiente
```bash
cp env.example .env
# Edite o .env com suas configurações
```

#### 3. Iniciar Serviços por Ambiente

```bash
# Development (padrão)
ENVIRONMENT=dev docker-compose up -d

# Staging
ENVIRONMENT=staging docker-compose up -d

# Production
ENVIRONMENT=production docker-compose --env-file .env.production.local up -d
```

#### 4. Executar a API (opcional, sem Docker)

```bash
# Configurar as variáveis de ambiente primeiro
export JWT_SECRET=minha-chave-secreta-super-segura
export SERVER_PORT=1324
export GOFF_ENDPOINT=http://localhost:1031
export ENVIRONMENT=dev

# Executar
go run src/cmd/server/main.go
```

### Outros comandos úteis

```bash
make help           # Lista todos os comandos
make dev-up         # Inicia development
make staging-up     # Inicia staging
make prod-up        # Inicia production
make test-flags     # Testa se está funcionando
make backup-flags   # Backup das flags
make clean          # Remove tudo
```

## Endpoints

### Health Checks

#### Liveness Probe
```bash
GET /health
```
Verifica se a aplicação está rodando.

**Response:**
```json
{
  "status": "ok"
}
```

#### Readiness Probe
```bash
GET /ready
```
Verifica se a aplicação está pronta para receber tráfego (checa conectividade com GO Feature Flag).

**Response (pronto):**
```json
{
  "status": "ready"
}
```

**Response (não pronto):**
```json
{
  "status": "unavailable",
  "message": "GO Feature Flag service is not available"
}
```

### Feature Flags

#### Obter Flags
```bash
GET /api/v1/flags
```

**Headers esperados do Flutter:**
- `Authorization: Bearer <token>` (opcional - se o usuário estiver autenticado)
- `X-App-Version: 1.2.0` (versão do app)
- `X-Platform: android` ou `ios`
- `X-Device-ID: <uuid>` (se não autenticado, usar device ID)

**Exemplos:**

Usuário autenticado:
```bash
curl -X GET http://localhost:1324/api/v1/flags \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-App-Version: 1.2.0" \
  -H "X-Platform: android"
```

Usuário anônimo:
```bash
curl -X GET http://localhost:1324/api/v1/flags \
  -H "X-App-Version: 1.2.0" \
  -H "X-Platform: ios" \
  -H "X-Device-ID: 550e8400-e29b-41d4-a716-446655440000"
```

**Response:**
```json
{
  "flags": {
    "front-dark-mode": false,
    "maintenance_mode": false,
    "maintenance_title": "Título da Manutenção",
    "maintenance_message": "Mensagem da manutenção",
    "feedback_enabled": true,
    "force_update_enabled": false,
    "minimum_app_version": "1.0.0",
    "update_title": "Atualização Necessária",
    "update_message": "Atualize para continuar",
    "new_dashboard": true
  }
}
```

**Tipos de valores suportados:**
- `boolean` - true/false
- `string` - texto
- `number` - inteiro ou decimal
- `object` - JSON object
- `array` - JSON array

## Configuração das Feature Flags

As flags são configuradas nos arquivos YAML em `/flags`:

- `front.yaml` - Flags do frontend/mobile
- `api.yaml` - Flags da API
- `shared.yaml` - Flags compartilhadas

### Exemplo de targeting por usuário:
```yaml
feedback_enabled:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: targetingKey eq "user-id-123"
      variation: disabled
  defaultRule:
    variation: enabled
```

### Exemplo de targeting por versão do app:
```yaml
force_update_enabled:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: app_version lt "1.2.0"
      variation: enabled
  defaultRule:
    variation: disabled
```

### Exemplo de targeting por plataforma:
```yaml
new_feature:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: platform eq "android"
      variation: enabled
  defaultRule:
    variation: disabled
```

## Integração Flutter

### Exemplo de código Flutter:

```dart
import 'package:http/http.dart' as http;
import 'dart:convert';

class FeatureFlagService {
  final String baseUrl = 'http://localhost:1324';
  
  Future<Map<String, dynamic>> getFlags({String? authToken}) async {
    final headers = {
      'X-App-Version': '1.2.0',
      'X-Platform': Platform.isAndroid ? 'android' : 'ios',
    };
    
    if (authToken != null) {
      headers['Authorization'] = 'Bearer $authToken';
    } else {
      headers['X-Device-ID'] = await getDeviceId();
    }
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/flags'),
      headers: headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['flags'] as Map<String, dynamic>;
    } else {
      throw Exception('Failed to load flags');
    }
  }
}

// Uso no app
final flags = await FeatureFlagService().getFlags();

if (flags['maintenance_mode'] == true) {
  showMaintenanceScreen();
}

if (flags['new_dashboard'] == true) {
  Navigator.push(context, NewDashboardRoute());
}

final minVersion = flags['minimum_app_version'] as String;
if (needsUpdate(currentVersion, minVersion)) {
  showUpdateDialog();
}
```

## Arquitetura Multi-Aplicação

Este projeto serve como **centro único** para gerenciar flags de múltiplas aplicações:

```
┌─────────────┐
│   Flutter   │ ──HTTP──> Flag API (:1324) ──SDK──> Relay Proxy (:1031)
└─────────────┘

┌─────────────┐
│  User API   │ ──SDK──────────────────────────> Relay Proxy (:1031)
└─────────────┘

┌─────────────┐
│ Payment API │ ──SDK──────────────────────────> Relay Proxy (:1031)
└─────────────┘
```

**Como funciona:**
1. Relay Proxy lê flags de `flags/{app}/{env}.yaml`
2. Flutter chama Flag API via HTTP (bulk evaluation)
3. Outras APIs Go usam SDK OpenFeature direto (on-demand)
4. Mudanças nos YAMLs são detectadas automaticamente

Veja `DEPLOYMENT.md` para guia completo de deploy por ambiente.

## Desenvolvimento

### Estrutura de logs

Os logs são emitidos em formato JSON estruturado:

```json
{
  "time": "2025-10-15T10:30:00Z",
  "level": "INFO",
  "msg": "request",
  "method": "GET",
  "path": "/api/v1/flags",
  "status": 200,
  "latency": 15000000,
  "ip": "127.0.0.1"
}
```

### Adicionando novas flags

Para adicionar uma nova flag:

#### 1. Adicione no arquivo YAML

```yaml
# flags/front.yaml
nova_feature:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: disabled
```

#### 2. Adicione no código Go

```go
// src/internal/services/featureflag.go
func (s *FeatureFlagService) EvaluateFlags(ctx, clientCtx) {
    // ... flags existentes
    
    // Adicione sua nova flag
    flags["nova_feature"], _ = s.client.BooleanValue(ctx, "nova_feature", false, evalCtx)
}
```

#### 3. Reinicie o servidor

A flag estará disponível no endpoint `/api/v1/flags`.

**Nota:** Mudanças no YAML são recarregadas automaticamente pelo GO Feature Flag a cada 1s, mas para adicionar uma nova flag no código, é necessário reiniciar o servidor.

Veja `flags/README.md` para exemplos de targeting e configuração avançada de flags.

## Produção

### Checklist de segurança:

- [ ] Usar `JWT_SECRET` forte e aleatório
- [ ] Configurar `CORS` com domínios específicos em `src/internal/middleware/cors.go`
- [ ] Usar HTTPS
- [ ] Configurar rate limiting (adicionar middleware)
- [ ] Monitorar logs e métricas
- [ ] Configurar health checks no orquestrador (Kubernetes, ECS, etc)

### Docker

```dockerfile
# Exemplo de Dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o server src/cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 1324
CMD ["./server"]
```

## Licença

MIT
