# Guia de Deploy por Ambiente

Este guia explica como fazer deploy do Better Feature Flag em diferentes ambientes.

## Ambientes Disponíveis

- **Development** - Para desenvolvimento local
- **Staging** - Para testes e homologação
- **Production** - Ambiente de produção

## Pré-requisitos

- Docker & Docker Compose
- Go 1.23+ (para compilar o binário)
- Acesso aos secrets de cada ambiente

---

## Development

### 1. Configurar Variáveis

Não precisa criar `.env`, já tem valores padrão seguros para dev.

Opcionalmente, pode criar `.env.dev.local` (não vai pro git):
```env
JWT_SECRET=dev-local-secret
```

### 2. Iniciar Serviços

```bash
# Usar arquivo env padrão (dev)
docker-compose up

# Ou especificar explicitamente
ENVIRONMENT=dev docker-compose up
```

### 3. Testar

```bash
# Health check
curl http://localhost:1031/health

# Flag API
curl http://localhost:1324/health

# Buscar flags
curl -H "X-App-Version: 1.0.0" -H "X-Platform: android" \
  http://localhost:1324/api/v1/flags
```

### 4. Ver Logs

```bash
# Todos os logs
docker-compose logs -f

# Apenas relay proxy
docker-compose logs -f go-feature-flag

# Apenas flag API
docker-compose logs -f flag-api
```

---

## Staging

### 1. Configurar Secrets

Crie `.env.staging.local` com valores reais:
```env
ENVIRONMENT=staging
JWT_SECRET=staging-real-secret-here
```

### 2. Build da Imagem

```bash
# Build da flag API
docker build -t better-feature-flag-api:staging .
```

### 3. Deploy

```bash
# Usando docker-compose
docker-compose --env-file .env.staging.local up -d

# Ou exportando variáveis
export ENVIRONMENT=staging
export JWT_SECRET=staging-secret
docker-compose up -d
```

### 4. Validar

```bash
# Verifica se está usando arquivos corretos
docker-compose exec go-feature-flag cat /goff/goff-proxy.yaml

# Deve mostrar paths com "staging"
# /goff/flags/flutter/staging.yaml
# /goff/flags/shared/staging.yaml
```

---

## Production

### 1. Preparar Secrets

**IMPORTANTE:** Nunca commite secrets de produção!

Crie `.env.production.local`:
```env
ENVIRONMENT=production
JWT_SECRET=super-secure-production-secret-change-this
```

### 2. Build Otimizado

```bash
# Build com otimizações
docker build \
  --build-arg GO_VERSION=1.23 \
  --build-arg GOOS=linux \
  --build-arg GOARCH=amd64 \
  -t better-feature-flag-api:production \
  .
```

### 3. Deploy

#### Opção A: Docker Compose (Servidor Simples)

```bash
docker-compose --env-file .env.production.local up -d
```

#### Opção B: Kubernetes (Recomendado)

```yaml
# kubernetes/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-feature-flag
spec:
  replicas: 2
  selector:
    matchLabels:
      app: go-feature-flag
  template:
    metadata:
      labels:
        app: go-feature-flag
    spec:
      containers:
      - name: relay-proxy
        image: gofeatureflag/go-feature-flag:latest
        ports:
        - containerPort: 1031
        volumeMounts:
        - name: flags
          mountPath: /goff/flags
        - name: config
          mountPath: /goff/goff-proxy.yaml
          subPath: goff-proxy.yaml
        env:
        - name: ENVIRONMENT
          value: "production"
      volumes:
      - name: flags
        configMap:
          name: feature-flags
      - name: config
        configMap:
          name: goff-config
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flag-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flag-api
  template:
    metadata:
      labels:
        app: flag-api
    spec:
      containers:
      - name: api
        image: better-feature-flag-api:production
        ports:
        - containerPort: 1324
        env:
        - name: GOFF_ENDPOINT
          value: "http://go-feature-flag:1031"
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: flag-api-secrets
              key: jwt-secret
        - name: ENVIRONMENT
          value: "production"
        livenessProbe:
          httpGet:
            path: /health
            port: 1324
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 1324
          initialDelaySeconds: 5
          periodSeconds: 5
```

Deploy no Kubernetes:
```bash
kubectl apply -f kubernetes/
```

### 4. Validar Produção

```bash
# Health checks
curl https://flags.yourcompany.com/health
curl https://flags.yourcompany.com/ready

# Testa flag API
curl -H "X-App-Version: 1.0.0" -H "X-Platform: ios" \
  https://flags.yourcompany.com/api/v1/flags
```

### 5. Monitoramento

```bash
# Prometheus metrics (se configurado)
curl https://flags.yourcompany.com/metrics

# Logs
docker-compose logs -f --tail=100
# ou
kubectl logs -f deployment/flag-api
```

---

## Rollback

### Development
```bash
git checkout HEAD~1
docker-compose restart
```

### Staging/Production
```bash
# Docker Compose
docker-compose down
git checkout <commit-anterior>
docker-compose up -d

# Kubernetes
kubectl rollout undo deployment/flag-api
kubectl rollout undo deployment/go-feature-flag
```

---

## Mudança de Flags (Sem Deploy)

### Para Mudar Valores de Flags

```bash
# 1. Edita o arquivo
vim flags/flutter/production.yaml

# 2. Commit
git add flags/
git commit -m "Enable new feature in production"
git push

# 3. Relay proxy detecta automaticamente (1-5s)
# Não precisa reiniciar nada!
```

### Para Adicionar Nova Flag

Se a flag **já existe no código**:
```bash
# Apenas adiciona no YAML
vim flags/flutter/production.yaml
git commit && git push
# Detectado automaticamente
```

Se é uma **flag nova no código**:
```bash
# 1. Adiciona no código Go
vim src/internal/services/featureflag.go

# 2. Adiciona no YAML
vim flags/flutter/production.yaml

# 3. Build e deploy
docker build -t flag-api:new-version .
kubectl set image deployment/flag-api api=flag-api:new-version
```

---

## Configuração de DNS

### Development
```
localhost:1324
```

### Staging
```
flags-staging.yourcompany.com → Load Balancer → flag-api pods
```

### Production
```
flags.yourcompany.com → Load Balancer → flag-api pods (multi-region)
```

---

## Backup de Flags

```bash
# Backup manual
tar -czf flags-backup-$(date +%Y%m%d).tar.gz flags/

# Ou usar Git (recomendado)
git add flags/
git commit -m "Backup flags $(date)"
git push
```

---

## Troubleshooting

### Relay Proxy não inicia

```bash
# Verifica logs
docker-compose logs go-feature-flag

# Verifica se config está correto
docker-compose exec go-feature-flag cat /goff/goff-proxy.yaml

# Testa manualmente
docker run -v $(pwd)/flags:/goff/flags \
  -v $(pwd)/goff-proxy-dev.yaml:/goff/goff-proxy.yaml \
  -p 1031:1031 \
  gofeatureflag/go-feature-flag:latest
```

### Flag API retorna 500

```bash
# Verifica se relay proxy está acessível
docker-compose exec flag-api wget -O- http://go-feature-flag:1031/health

# Verifica variáveis de ambiente
docker-compose exec flag-api env | grep GOFF
```

### Flags não estão atualizando

```bash
# Força restart do relay proxy
docker-compose restart go-feature-flag

# Verifica polling interval
cat goff-proxy-production.yaml | grep polling
```

---

## Checklist de Produção

Antes de ir para produção:

- [ ] JWT_SECRET forte e aleatório
- [ ] CORS configurado com domínios específicos
- [ ] HTTPS configurado
- [ ] Health checks no Load Balancer
- [ ] Monitoramento configurado (Prometheus/Grafana)
- [ ] Logs centralizados (ELK/Datadog)
- [ ] Backup automático dos YAMLs
- [ ] Rate limiting configurado
- [ ] Alertas configurados (PagerDuty/Opsgenie)
- [ ] Documentação atualizada
- [ ] Runbook de incidentes criado

