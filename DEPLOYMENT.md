# Guia de Deploy

Este guia explica como subir o **GO Feature Flag relay-proxy** e a **Flag API**.

Não existe mais separação de flags por ambiente (dev/prod) dentro desse repositório: o relay usa um único `goff-proxy.yaml` e os arquivos em `flags/*.yaml`.

## Pré-requisitos

- Docker & Docker Compose
- Go 1.23+ (para compilar o binário)
- Acesso aos secrets de cada ambiente

---

## Subir localmente (Docker Compose)

```bash
docker-compose up
```

## Testar

```bash
# Health check
curl http://localhost:1031/health

# Flag API
curl http://localhost:1324/health

# Buscar flags
curl -H "X-App-Version: 1.0.0" -H "X-Platform: android" \
  http://localhost:1324/api/v1/flags
```

## Ver logs

```bash
# Todos os logs
docker-compose logs -f

# Apenas relay proxy
docker-compose logs -f go-feature-flag

# Apenas flag API
docker-compose logs -f flag-api
```

---

## Rollback

### Development
```bash
git checkout HEAD~1
docker-compose restart
```

### Production
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
vim flags/flutter.yaml

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
vim flags/flutter.yaml
git commit && git push
# Detectado automaticamente
```

Se é uma **flag nova no código**:
```bash
# 1. Adiciona no código Go
vim src/internal/services/featureflag.go

# 2. Adiciona no YAML
vim flags/flutter.yaml

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
  -v $(pwd)/goff-proxy.yaml:/goff/goff-proxy.yaml \
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

