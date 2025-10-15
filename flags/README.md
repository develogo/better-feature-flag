# Feature Flags Configuration

Este diretório contém os arquivos YAML com as definições das feature flags.

## Arquivos

- `front.yaml` - Flags para o frontend/mobile
- `api.yaml` - Flags para a API backend
- `shared.yaml` - Flags compartilhadas entre frontend e backend

## Como Adicionar Uma Nova Flag

**Não é necessário modificar código Go!** Apenas adicione a flag no arquivo YAML apropriado e ela será automaticamente avaliada e retornada pela API.

### Exemplo: Flag Booleana

```yaml
nova_feature_enabled:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    variation: disabled
```

### Exemplo: Flag String

```yaml
welcome_message:
  variations:
    default: "Bem-vindo!"
    special: "Bem-vindo, usuário especial!"
  defaultRule:
    variation: default
```

### Exemplo: Flag Numérica

```yaml
max_items_per_page:
  variations:
    small: 10
    medium: 25
    large: 50
  defaultRule:
    variation: medium
```

## Targeting (Direcionamento)

### Por Usuário Específico

```yaml
beta_feature:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: targetingKey eq "user-id-123"
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
android_only_feature:
  variations:
    enabled: true
    disabled: false
  targeting:
    - query: platform eq "android"
      variation: enabled
  defaultRule:
    variation: disabled
```

### Múltiplas Condições

```yaml
premium_feature:
  variations:
    enabled: true
    disabled: false
  targeting:
    # Usuários premium no Android com app >= 2.0
    - query: platform eq "android" AND app_version gte "2.0.0" AND user_id contains "premium"
      variation: enabled
    # Todos os usuários no iOS
    - query: platform eq "ios"
      variation: enabled
  defaultRule:
    variation: disabled
```

## Atributos Disponíveis para Targeting

Estes atributos são automaticamente enviados pelo serviço:

- `targetingKey` - ID do usuário (se autenticado) ou device ID
- `app_version` - Versão do app (header `X-App-Version`)
- `platform` - Plataforma: "android" ou "ios" (header `X-Platform`)
- `user_id` - ID do usuário (se autenticado)
- `email` - Email do usuário (se autenticado)
- `username` - Username do usuário (se autenticado)
- `device_id` - ID do dispositivo (se não autenticado)

## Operadores Disponíveis

- `eq` - Igual
- `ne` - Diferente
- `lt` - Menor que
- `lte` - Menor ou igual
- `gt` - Maior que
- `gte` - Maior ou igual
- `contains` - Contém (strings)
- `startsWith` - Começa com
- `endsWith` - Termina com
- `matches` - Regex
- `AND` - E lógico
- `OR` - Ou lógico

## Rollout Progressivo

```yaml
gradual_rollout:
  variations:
    enabled: true
    disabled: false
  defaultRule:
    percentage:
      enabled: 50  # 50% dos usuários verão habilitado
      disabled: 50
```

## Reload Automático

O GO Feature Flag verifica mudanças nos arquivos YAML automaticamente a cada segundo (configurado em `goff-proxy.yaml`). Não é necessário reiniciar o serviço para aplicar mudanças.

## Testando Flags

```bash
# Usuário anônimo
curl -X GET http://localhost:1324/api/v1/flags \
  -H "X-App-Version: 2.0.0" \
  -H "X-Platform: android" \
  -H "X-Device-ID: device-123"

# Usuário autenticado
curl -X GET http://localhost:1324/api/v1/flags \
  -H "Authorization: Bearer SEU_TOKEN" \
  -H "X-App-Version: 2.0.0" \
  -H "X-Platform: ios"
```

## Documentação Oficial

Para mais detalhes sobre configuração de flags, consulte:
- [GO Feature Flag Documentation](https://gofeatureflag.org/)
- [Targeting Rules](https://gofeatureflag.org/docs/configure_flag/flag_format)

