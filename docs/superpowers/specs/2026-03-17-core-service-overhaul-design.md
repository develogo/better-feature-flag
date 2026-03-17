# Better Feature Flag — Core Service Overhaul

## Context

The `better-feature-flag` service is a feature flag proxy server for BetterCity. It sits between the Flutter mobile app and a GO Feature Flag (GOFF) relay proxy, providing bulk flag evaluation with Keycloak authentication and device-context targeting.

The goal of this overhaul is to transform it into a **core service** ready to support multiple BetterCity applications. The Flutter app remains the only REST consumer; backend services will consume the GOFF relay directly. The improvements span code quality, testability, security, observability, and infrastructure.

**Scope:** Ambitious restructure — breaking changes to the API are acceptable.

## Problems Addressed

| # | Area | Problem |
|---|------|---------|
| 1 | Code | Zero test files in the entire project |
| 2 | Code | No interfaces — all services are concrete structs, untestable and non-swappable |
| 3 | Code | Flag definitions hardcoded in Go (`FrontendFlagDefinitions` array) — adding a flag requires code change + deploy |
| 4 | Code | Naming inconsistency — `dark-mode` (kebab) in GOFF YAML vs `dark_mode` (snake) in Go |
| 5 | Code | Config `log_level` field is ignored — logger always uses `slog.LevelInfo` |
| 6 | Code | Only `bool` and `string` flag types supported — OpenFeature supports `int`/`float` too |
| 7 | Code | No request ID / tracing — impossible to correlate logs across a single request |
| 8 | Security | Keycloak `client_secret` committed in `config/local.yaml` |
| 9 | Repo | Compiled binaries (`bin/server`, `bin/server.exe`) committed to git |
| 10 | Security | CORS `AllowOrigins: ["*"]` in all environments including production |
| 11 | Security | No rate limiting — no protection against abuse |
| 12 | CI | Pipeline only builds and pushes images — no tests run |
| 13 | Infra | Docker Compose requires manually pre-created external network `bettercity_local` |
| 14 | Infra | No OpenAPI/Swagger contract documentation |
| 15 | Infra | `shared.yaml` flags exist but no endpoint serves them |
| 16 | Infra | No structured organization for multi-app flag files |

## Design

### 1. Interfaces and Dependency Inversion

All services currently expose concrete structs. Handlers and middleware depend directly on `*services.FeatureFlagService` and `*services.KeycloakService`.

**Change:** Define interfaces that handlers and middleware depend on:

```go
// internal/services/interfaces.go

type FeatureFlagEvaluator interface {
    EvaluateFlags(ctx context.Context, flags []models.FlagDefinition, clientCtx *models.ClientContext) (map[string]interface{}, error)
    HealthCheck(ctx context.Context, flags []models.FlagDefinition) error
}

type TokenValidator interface {
    ValidateToken(ctx context.Context, token string) (*models.TokenClaims, error)
}

type FlagRegistry interface {
    GetFlagsForApp(appName string) ([]models.FlagDefinition, error)
}
```

- Handlers and middleware depend on **interfaces**, not structs
- Current implementations (`FeatureFlagService`, `KeycloakService`, `FlagRegistryService`) satisfy these interfaces implicitly
- Tests use mocks implementing the same interfaces
- FX continues injecting the concrete implementations; the contract is the interface

**Shared types in `internal/models/`:**
- `TokenClaims` moves from `internal/services/keycloak.go` to `internal/models/token.go` — avoids circular dependency between middleware and services
- `FlagDefinition` and `FlagValueType` move from `internal/services/frontend_flags.go` to `internal/models/flag.go` — shared between registry and evaluator

**Design note:** The handler orchestrates flag evaluation by (1) getting flag definitions from `FlagRegistry.GetFlagsForApp(appName)`, then (2) passing them to `FeatureFlagEvaluator.EvaluateFlags(ctx, flags, clientCtx)`. This keeps the evaluator decoupled from the registry — it evaluates whatever flags it receives.

**`HealthCheck` note:** The current implementation hardcodes `maintenance_mode` as the probe flag. The new interface accepts `flags []models.FlagDefinition` — the health handler fetches any app's flags from the registry (e.g., the first registered app) and passes them in. `HealthCheck` evaluates the first flag in the list as a connectivity probe. This way the health check is decoupled from any specific flag name and works as long as at least one flag exists in the registry.

**Files affected:**
- New: `internal/services/interfaces.go`, `internal/models/token.go`, `internal/models/flag.go`
- Modified: `internal/handlers/flags.go`, `internal/handlers/health.go`, `internal/middleware/auth.go`, `internal/services/featureflag.go` (method renamed from `EvaluateAllFlags` to `EvaluateFlags`, signature updated to accept `[]models.FlagDefinition`; `HealthCheck` updated to accept `[]models.FlagDefinition`), `internal/services/keycloak.go` (return type changes to `*models.TokenClaims`)

### 2. Config-Driven Flag Registry

`FrontendFlagDefinitions` is a hardcoded Go array. Adding a flag requires code changes and redeployment. Only `bool` and `string` types are supported.

**Change:** Move flag definitions to a YAML file loaded at startup:

```yaml
# config/flags.yaml
apps:
  flutter:
    flags:
      - name: dark_mode
        type: bool
        default: false
      - name: maintenance_mode
        type: bool
        default: false
      - name: feedback_enabled
        type: bool
        default: true
      - name: force_update_enabled
        type: bool
        default: false
      - name: minimum_app_version
        type: string
        default: "1.0.0"
```

**New `FlagRegistry` service:**
- Loads and validates the YAML at startup
- Supports 4 types: `bool`, `string`, `int`, `float`
- Organized by **app** — each app has its own flag set in the YAML
- Exposes methods: `GetFlagsForApp(appName string) ([]FlagDefinition, error)`, `GetAnyFlags() ([]FlagDefinition, error)` (used by health check)

**`config/flags.yaml` loading:** The file path is resolved via a new config field `app.flags_file` (default: `"config/flags.yaml"`). The registry uses the same Viper config path resolution as the main config (`./config`, `../config`, `../../config`), ensuring it works both locally (where the working directory is the repo root) and in Docker (where the working directory is `/app` and the config directory is copied to `/app/config`). The `Dockerfile.api` already copies the `config/` directory into the image, so `config/flags.yaml` will be available at runtime without changes to the Dockerfile.

**Endpoint change:**
- `GET /api/v1/flags` accepts query param `?app=flutter`
- Default: `flutter` (backwards-compatible for existing clients during migration)
- Returns 400 if app name is unknown

**Startup validation:**
- Flags defined in the registry but absent in GOFF emit a warning log

**Naming convention:** All flag names use **snake_case** as the canonical format. The registry flag names must match the GOFF YAML flag keys exactly. All kebab-case flag names across all GOFF YAML files must be renamed to snake_case:

- `flags/apps/flutter.yaml`: `dark-mode` → `dark_mode`
- `flags/shared.yaml`: `maintenance-mode` → `maintenance_mode`, `redis-cache-enabled` → `redis_cache_enabled`, `log-level` → `log_level`, `rate-limit-enabled` → `rate_limit_enabled`

**Files affected:**
- New: `config/flags.yaml`, `internal/services/registry.go`
- Removed: `internal/services/frontend_flags.go`
- Modified: `internal/services/featureflag.go`, `internal/handlers/flags.go`, `flags/apps/flutter.yaml`, `flags/shared.yaml`

### 3. Tests and CI

**Testing strategy using `testify`:**

| Layer | What to test | Approach |
|-------|-------------|----------|
| Services | `FeatureFlagService` — flag evaluation by type, fallback to defaults, authenticated vs anonymous context | Unit tests with mocked OpenFeature client |
| Services | `KeycloakService` — active/inactive token, malformed JWT, claims extraction | Unit tests with mocked gocloak client |
| Services | `FlagRegistry` — YAML parsing, type validation, unknown app | Unit tests with test YAML fixtures |
| Middleware | `OptionalJWT` — with/without Bearer token, device headers, invalid token | Unit tests with mocked `TokenValidator` |
| Handlers | `GET /api/v1/flags` — full request with mocked `FeatureFlagEvaluator` | Integration tests using Echo test server |
| Handlers | `GET /health`, `GET /ready` — responses and GOFF unavailable scenario | Integration tests using Echo test server |

**CI pipeline update:**

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go test ./... -race -coverprofile=coverage.out
      - run: go vet ./...

  relay:
    needs: test
    # ... existing build job

  api:
    needs: test
    # ... existing build job
```

- Insert `test` job before existing `relay` and `api` jobs; existing job definitions remain unchanged except adding `needs: test`
- The existing `git-actions` checkout and composite action usage in `relay`/`api` jobs is preserved as-is
- `-race` flag detects race conditions
- `go vet` catches static analysis issues

### 4. Observability and Security

**4a. Request ID / Tracing:**
- New middleware generates or propagates `X-Request-ID` on each request
- ID included in all logs for that request via `slog.With(slog.String("request_id", id))`
- Enables log correlation across services

**4b. Log level from config:**
- Add `LogLevel string` field to `AppConfig` struct (currently only has `Port`)
- `ProvideLogger` changes signature to accept `*config.Config` and reads `app.log_level`
- Applies parsed level to `slog.HandlerOptions.Level`
- Fixes the current behavior where `log_level: "debug"` is ignored

**4c. Secrets out of git:**
- Remove `client_secret` value from `config/local.yaml`
- Use `.env` + Viper for secrets in development (`.env` already in `.gitignore`)
- `local.yaml` keeps only non-sensitive values, with a placeholder comment

**4d. Configurable CORS:**
- Move allowed origins to config YAML per environment:

```yaml
# config/local.yaml
app:
  cors_origins: ["*"]

# config/production.yaml
app:
  cors_origins:
    - "https://app.bettercity.app"
    - "https://admin.bettercity.app"
```

- **Fix CORS `AllowHeaders` mismatch:** The current `AllowHeaders` list uses `X-App-Version`, `X-Platform`, `X-Device-ID`, but the `OptionalJWT` middleware actually reads: `Device-ID`, `Platform`, `Platform-Version`, `Device-Model`, `Device-Architecture`, `Device-Brand`, `Mobile`, `Device`, `App-Name`, `App-Version`, `Package-Name`, `Build-Number`. The CORS config must be updated to allow all headers that the auth middleware consumes, otherwise CORS preflight will block them.

**4e. Rate limiting:**
- New middleware using in-memory rate limiter (token bucket per IP)
- Configurable in YAML: `app.rate_limit` (requests per second, default: 100 if not set)
- Applied to `/api/v1/*` routes
- State is in-memory and resets on restart (acceptable for this service)
- Uses `c.RealIP()` which respects `X-Forwarded-For` / `X-Real-IP` headers — correct behavior behind a reverse proxy in production

**4f. Binaries out of git:**
- Add `bin/` to `.gitignore`
- Remove `bin/server` and `bin/server.exe` from git tracking

**Files affected:**
- New: `internal/middleware/requestid.go`, `internal/middleware/ratelimit.go`
- Modified: `internal/fx/fx.go` (logger), `internal/middleware/cors.go`, `internal/middleware/fx.go`, `internal/config/config.go`, `config/local.yaml`, `config/production.yaml`, `.gitignore`

### 5. Flag Files Organization and Docker

**Flag file structure:**

```
flags/
├── apps/
│   ├── flutter.yaml
│   ├── user-api.yaml      # future
│   └── payment-api.yaml   # future
└── shared.yaml
```

- Clear separation between app-specific and global flags
- `goff-proxy.yaml` updated to load `flags/apps/*.yaml` + `flags/shared.yaml`
- Each backend consuming the relay directly knows which flags are theirs
- **Shared flags resolution (Problem #15):** Shared flags are consumed directly by backend services via the GOFF relay SDK, not through this REST API. The directory reorganization makes this consumption pattern explicit — `flags/shared.yaml` is loaded by the relay and available to any GOFF SDK consumer, while this API only serves app-specific flags defined in `config/flags.yaml`

**Docker Compose for local dev:**

```yaml
services:
  relay:
    image: gofeatureflag/go-feature-flag:latest
    ports:
      - "1031:1031"
    volumes:
      - ./flags:/goff/flags
      - ./goff-proxy.yaml:/goff/goff-proxy.yaml

  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    ports:
      - "1324:80"
    environment:
      - APP_ENV=local
      - KEYCLOAK_CLIENT_SECRET=${KEYCLOAK_CLIENT_SECRET}
    depends_on:
      - relay

networks:
  default:
    name: bettercity_local
```

- No external network dependency — `make up` works out of the box
- The network is named `bettercity_local` so other BetterCity docker-compose files that reference it as external will still find it — this compose creates it if it doesn't exist
- API server included in compose for full local environment
- Secrets come from `.env` file

**Files affected:**
- Moved: `flags/flutter.yaml` → `flags/apps/flutter.yaml`
- Modified: `goff-proxy.yaml`, `docker-compose.yml`, `Makefile`

## Implementation Order

The sections above should be implemented in this order due to dependencies:

1. **Interfaces** (Section 1) — foundation for everything else
2. **Flag Registry** (Section 2) — depends on interfaces
3. **Observability & Security** (Section 4) — independent, but benefits from interfaces for testing
4. **Tests** (Section 3) — needs interfaces and registry in place to write meaningful tests
5. **Flag Files & Docker** (Section 5) — independent, can be done in parallel with tests

## Out of Scope

- OpenAPI/Swagger documentation — valuable but not critical for this phase
- Hot-reload of flag registry YAML — the path is open for future work
- Admin API for flag management — GOFF relay handles this
- Dashboard or UI — out of scope for a backend core service
