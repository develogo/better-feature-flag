# Feature Flags - Organização Multi-Aplicação e Ambientes

Este diretório contém as feature flags organizadas por **aplicação** e **ambiente**.

## Estrutura de Diretórios

```
flags/
├── flutter/                  # Flags para aplicativo Flutter
│   ├── dev.yaml             # Desenvolvimento
│   ├── staging.yaml         # Homologação
│   └── production.yaml      # Produção
│
├── user-api/                 # Flags para API de usuários
│   ├── dev.yaml
│   ├── staging.yaml
│   └── production.yaml
│
├── payment-api/              # Flags para API de pagamentos
│   ├── dev.yaml
│   ├── staging.yaml
│   └── production.yaml
│
└── shared/                   # Flags compartilhadas entre todas as apps
    ├── dev.yaml
    ├── staging.yaml
    └── production.yaml
```

## Como Funciona

### 1. Relay Proxy Carrega Flags por Ambiente

O relay proxy usa diferentes arquivos de configuração baseado na variável `ENVIRONMENT`:

```bash
# Development (padrão)
ENVIRONMENT=dev docker-compose up

# Staging
ENVIRONMENT=staging docker-compose up

# Production
ENVIRONMENT=production docker-compose up
```

Cada ambiente carrega apenas os arquivos correspondentes:
- `dev` → carrega todos os `*/dev.yaml`
- `staging` → carrega todos os `*/staging.yaml`
- `production` → carrega todos os `*/production.yaml`

### 2. Cada Aplicação Consume Suas Flags

**Flutter App:**
```dart
// Chama a Flag API que avalia flags de flutter/{env}.yaml
final flags = await FeatureFlagService().getAllFlags();
if (flags['maintenance_mode']) {
  showMaintenanceScreen();
}
```

**User API (Go):**
```go
// SDK conecta no relay proxy e avalia flags de user-api/{env}.yaml
enabled, _ := client.BooleanValue(ctx, "registration-enabled", true, evalCtx)
```

**Payment API (Go):**
```go
// SDK conecta no relay proxy e avalia flags de payment-api/{env}.yaml
useStripe, _ := client.BooleanValue(ctx, "payment-use-stripe", false, evalCtx)
```

**Flags Compartilhadas:**
```go
// Qualquer app pode acessar flags de shared/{env}.yaml
maintenanceMode, _ := client.BooleanValue(ctx, "maintenance-mode", false, evalCtx)
```

## Quando Usar Cada Diretório

### `flutter/`
Flags específicas para o app móvel:
- UI features (dark mode, new screens)
- App behavior (force update, feedback)
- Mobile-specific settings

### `user-api/`
Flags específicas para API de usuários:
- Registration settings
- Authentication methods
- User profile features

### `payment-api/`
Flags específicas para API de pagamentos:
- Payment providers (Stripe, PayPal)
- Transaction limits
- Refund policies

### `shared/`
Flags que afetam TODAS as aplicações:
- `maintenance-mode` - Desliga tudo
- `redis-cache-enabled` - Comportamento global de cache
- `log-level` - Nível de log em todos os serviços
- `rate-limit-enabled` - Rate limiting global

## Como Adicionar Uma Nova Flag

### Passo 1: Escolha o Diretório Correto

**Pergunta:** Onde essa flag será usada?
- Apenas no Flutter? → `flutter/{env}.yaml`
- Apenas na User API? → `user-api/{env}.yaml`
- Em múltiplas apps? → `shared/{env}.yaml`

### Passo 2: Adicione nos 3 Ambientes

```yaml
# flags/flutter/dev.yaml
nova_feature:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: enabled  # Habilitado em dev

# flags/flutter/staging.yaml
nova_feature:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: enabled  # Habilitado em staging

# flags/flutter/production.yaml
nova_feature:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: disabled  # Desabilitado em prod (rollout gradual)
```

### Passo 3: Se For Para Flutter, Adicione no Código

```go
// src/internal/services/featureflag.go
func (s *FeatureFlagService) EvaluateAllFlags(...) {
    // ... flags existentes
    
    flags["nova_feature"], _ = s.client.BooleanValue(ctx, "nova_feature", false, evalCtx)
}
```

### Passo 4: Reinicie o Relay Proxy (Se Necessário)

Mudanças nos YAMLs são detectadas automaticamente, mas se adicionar um novo arquivo:
```bash
docker-compose restart go-feature-flag
```

## Exemplos de Flags por Tipo

### Flag Booleana
```yaml
feature_enabled:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: disabled
```

### Flag String
```yaml
welcome_message:
  variations:
    default: "Bem-vindo!"
    premium: "Bem-vindo, usuário premium!"
  defaultRule:
    variation: default
```

### Flag Numérica
```yaml
max_items:
  variations:
    low: 10
    high: 100
  defaultRule:
    variation: low
```

## Targeting (Segmentação)

### Por Usuário Específico
```yaml
beta_feature:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: targetingKey eq "user-123"
      variation: enabled
  defaultRule:
    variation: disabled
```

### Por Versão do App
```yaml
new_ui:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: app_version gte "2.0.0"
      variation: enabled
  defaultRule:
    variation: disabled
```

### Por Plataforma
```yaml
android_feature:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: platform eq "android"
      variation: enabled
  defaultRule:
    variation: disabled
```

### Rollout Progressivo (Percentage)
```yaml
gradual_rollout:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    percentage:
      enabled: 10   # 10% dos usuários
      disabled: 90
```

## Estratégias de Rollout por Ambiente

### Development
- Todas as features habilitadas
- Targeting mínimo
- Valores permissivos

### Staging
- Algumas features habilitadas
- Targeting similar à produção
- Valores realistas

### Production
- Features desabilitadas por padrão
- Rollout gradual (percentage)
- Targeting específico
- Valores conservadores

## Atributos Disponíveis para Targeting

Automaticamente enviados pela aplicação:

**Flutter:**
- `targetingKey` - User ID ou Device ID
- `app_version` - Versão do app
- `platform` - "android" ou "ios"
- `user_id` - ID do usuário (se autenticado)
- `device_id` - ID do dispositivo (se não autenticado)

**APIs Go:**
- `targetingKey` - User ID
- `email` - Email do usuário
- `username` - Username
- Qualquer atributo customizado que você adicionar

## Operadores de Query

- `eq` - Igual
- `ne` - Diferente
- `lt` - Menor que
- `lte` - Menor ou igual
- `gt` - Maior que
- `gte` - Maior ou igual
- `contains` - Contém
- `startsWith` - Começa com
- `endsWith` - Termina com
- `matches` - Regex
- `AND` / `OR` - Combinação de condições

## Boas Práticas

### ✅ Faça

1. **Use nomes descritivos**
   - ✅ `registration-email-verification-enabled`
   - ❌ `flag1`

2. **Organize por aplicação**
   - Se usado só no Flutter → `flutter/`
   - Se usado só na User API → `user-api/`

3. **Mantenha consistência entre ambientes**
   - Mesmas flags nos 3 arquivos (dev, staging, prod)
   - Apenas variações nos valores

4. **Documente flags complexas**
   ```yaml
   # Esta flag controla o novo fluxo de onboarding
   # Habilitada apenas para usuários iOS >= 14
   new_onboarding:
     variations:
       enabled: true
       disabled: false
   ```

5. **Use percentage para rollout gradual**
   ```yaml
   defaultRule:
     percentage:
       enabled: 25  # Começa com 25%
       disabled: 75
   ```

### ❌ Evite

1. **Flags órfãs** - Remova flags não usadas
2. **Duplicação** - Não crie a mesma flag em múltiplos lugares
3. **Valores hardcoded** - Use flags mesmo em dev
4. **Targeting sem fallback** - Sempre tenha defaultRule

## Migração Entre Ambientes

### Promover flag de Dev → Staging
```bash
# 1. Teste em dev
# 2. Copie configuração para staging
cp flags/flutter/dev.yaml flags/flutter/staging.yaml

# 3. Ajuste valores se necessário
vim flags/flutter/staging.yaml

# 4. Deploy staging
ENVIRONMENT=staging docker-compose up
```

### Promover flag de Staging → Production
```bash
# 1. Teste em staging
# 2. Ajuste valores para produção (mais conservador)
vim flags/flutter/production.yaml

# 3. Deploy production
ENVIRONMENT=production docker-compose up
```

## Troubleshooting

### Flag não está sendo aplicada

1. Verifica se está no arquivo correto (`flutter/`, `user-api/`, etc)
2. Verifica se o ambiente está correto (`dev`, `staging`, `production`)
3. Verifica os logs do relay proxy:
   ```bash
   docker-compose logs go-feature-flag
   ```

### Mudança não está refletindo

1. Aguarde 1-5 segundos (polling interval)
2. Verifica se o arquivo está sendo montado corretamente:
   ```bash
   docker-compose exec go-feature-flag ls /goff/flags/
   ```

### Flag retorna sempre o valor padrão

1. Verifica targeting - pode estar bloqueando
2. Verifica `targetingKey` - pode estar incorreto
3. Adiciona logs na aplicação para debug

## Documentação Oficial

Para mais detalhes sobre configuração de flags:
- [GO Feature Flag Documentation](https://gofeatureflag.org/)
- [Targeting Rules](https://gofeatureflag.org/docs/configure_flag/flag_format)
- [OpenFeature Specification](https://openfeature.dev/)
