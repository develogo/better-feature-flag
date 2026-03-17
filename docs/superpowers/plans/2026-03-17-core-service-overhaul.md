# Core Service Overhaul Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform better-feature-flag from a hardcoded Flutter-only proxy into a testable, config-driven, multi-app core service.

**Architecture:** Introduce interfaces for all services, move flag definitions from hardcoded Go to YAML config, add observability (request IDs, configurable log levels), security hardening (CORS, rate limiting, secrets), comprehensive tests, and CI test gate. Reorganize flag files and Docker for multi-app support.

**Tech Stack:** Go 1.23, Echo v4, Uber FX, OpenFeature SDK, GO Feature Flag, Keycloak (gocloak), Viper, testify

**Spec:** `docs/superpowers/specs/2026-03-17-core-service-overhaul-design.md`

---

### Task 1: Shared model types — TokenClaims and FlagDefinition

> **Note:** Tasks 1, 2, and 3 form an atomic unit. The code will NOT compile after Task 1 or Task 2 alone because handlers/middleware reference old method names and the FlagRegistry has no implementation yet. This is intentional — each task changes one layer. The code compiles fully after Task 3. If you are an agentic worker, **proceed through all three tasks before expecting a passing build.**

Move `TokenClaims` from `internal/services/keycloak.go` and `FlagDefinition`/`FlagValueType` from `internal/services/frontend_flags.go` into `internal/models/`. This unblocks interfaces (Task 2) by putting shared types in a dependency-free package.

**Files:**
- Create: `internal/models/token.go`
- Create: `internal/models/flag.go`
- Modify: `internal/services/keycloak.go:22-27` (remove `TokenClaims` struct)
- Modify: `internal/services/keycloak.go:49,81,100` (update references to `models.TokenClaims`)
- Modify: `internal/services/featureflag.go:40,43-96` (update references to `models.FlagDefinition`, `models.FlagValueType*`)
- Delete: `internal/services/frontend_flags.go`

- [ ] **Step 1: Create `internal/models/token.go`**

```go
package models

type TokenClaims struct {
	Sub      string
	Email    string
	Username string
	Active   bool
}
```

- [ ] **Step 2: Create `internal/models/flag.go`**

```go
package models

type FlagValueType string

const (
	FlagValueTypeBool   FlagValueType = "bool"
	FlagValueTypeString FlagValueType = "string"
	FlagValueTypeInt    FlagValueType = "int"
	FlagValueTypeFloat  FlagValueType = "float"
)

type FlagDefinition struct {
	Name    string `yaml:"name"`
	Type    FlagValueType `yaml:"type"`
	Default any `yaml:"default"`
}
```

- [ ] **Step 3: Update `internal/services/keycloak.go`**

Remove the `TokenClaims` struct (lines 22-27). Add `"better-feature-flag/internal/models"` to imports. Change all `*TokenClaims` references to `*models.TokenClaims`:
- Line 49: `func (s *KeycloakService) ValidateToken(...) (*models.TokenClaims, error)`
- Line 81: `func (s *KeycloakService) extractClaimsFromJWT(...) (*models.TokenClaims, error)`
- Line 100: `return &models.TokenClaims{`

Keep `JWTClaims` struct in keycloak.go — it is an internal implementation detail of JWT parsing.

- [ ] **Step 4: Update `internal/services/featureflag.go`**

Replace `FrontendFlagDefinitions` iteration with the new `flagDefs` parameter. Update all type references to use `models.*` prefix (`models.FlagValueTypeBool`, `models.FlagValueTypeString`, etc.).

Change the method signature from:
```go
func (s *FeatureFlagService) EvaluateAllFlags(ctx context.Context, clientCtx *models.ClientContext) (map[string]interface{}, error) {
```
to:
```go
func (s *FeatureFlagService) EvaluateFlags(ctx context.Context, flagDefs []models.FlagDefinition, clientCtx *models.ClientContext) (map[string]interface{}, error) {
```

Update the method body:
- Line 40: `flags := make(map[string]interface{}, len(flagDefs))`
- Line 43: `for _, def := range flagDefs {`
- Line 45: `case models.FlagValueTypeBool:`
- Line 67: `case models.FlagValueTypeString:`

Add `int` and `float` cases after the `string` case:

```go
		case models.FlagValueTypeInt:
			var defaultInt64 int64
			switch v := def.Default.(type) {
			case int:
				defaultInt64 = int64(v)
			case int64:
				defaultInt64 = v
			case float64:
				// YAML may parse integers as float64
				defaultInt64 = int64(v)
			default:
				defaultedCount++
				flags[def.Name] = int64(0)
				s.logger.Warn("invalid int flag default; using 0",
					slog.String("flag", def.Name),
				)
				continue
			}
			value, err := s.client.IntValue(ctx, def.Name, defaultInt64, evalCtx)
			if err != nil {
				defaultedCount++
				flags[def.Name] = defaultInt64
				s.logger.Warn("flag evaluation failed; using default",
					slog.String("flag", def.Name),
					slog.String("type", string(def.Type)),
					slog.String("error", err.Error()),
				)
				continue
			}
			flags[def.Name] = value
		case models.FlagValueTypeFloat:
			defaultValue, ok := def.Default.(float64)
			if !ok {
				defaultedCount++
				flags[def.Name] = 0.0
				s.logger.Warn("invalid float flag default; using 0.0",
					slog.String("flag", def.Name),
				)
				continue
			}
			value, err := s.client.FloatValue(ctx, def.Name, defaultValue, evalCtx)
			if err != nil {
				defaultedCount++
				flags[def.Name] = defaultValue
				s.logger.Warn("flag evaluation failed; using default",
					slog.String("flag", def.Name),
					slog.String("type", string(def.Type)),
					slog.String("error", err.Error()),
				)
				continue
			}
			flags[def.Name] = value
```

Update `HealthCheck` signature:
```go
func (s *FeatureFlagService) HealthCheck(ctx context.Context, flags []models.FlagDefinition) error {
	if len(flags) == 0 {
		return fmt.Errorf("no flags available for health check")
	}
	evalCtx := of.NewEvaluationContext("health-check", map[string]interface{}{})
	_, err := s.client.BooleanValue(ctx, flags[0].Name, false, evalCtx)
	return err
}
```

- [ ] **Step 5: Delete `internal/services/frontend_flags.go`**

Remove the entire file. The `FrontendFlagDefinitions` variable and the old `FlagDefinition`/`FlagValueType` types are now replaced.

- [ ] **Step 6: Verify compilation**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go build ./...`

Expected: Compilation errors in handlers (they still reference old method names). This is expected — handlers will be updated in Task 2.

- [ ] **Step 7: Commit**

```bash
git add internal/models/token.go internal/models/flag.go internal/services/keycloak.go internal/services/featureflag.go
git rm internal/services/frontend_flags.go
git commit -m "refactor: move shared types to models, update service signatures

Move TokenClaims to models/token.go and FlagDefinition to models/flag.go.
Rename EvaluateAllFlags to EvaluateFlags accepting flag definitions parameter.
Add int/float flag type support. Update HealthCheck to accept flags."
```

---

### Task 2: Service interfaces

Define the three interfaces and update all consumers (handlers, middleware, FX modules) to depend on interfaces instead of concrete types.

**Files:**
- Create: `internal/services/interfaces.go`
- Modify: `internal/handlers/flags.go` (depend on interfaces)
- Modify: `internal/handlers/health.go` (depend on interfaces)
- Modify: `internal/handlers/fx.go` (use fx.As for interface binding)
- Modify: `internal/middleware/auth.go` (depend on `TokenValidator` interface)
- Modify: `internal/middleware/fx.go` (use fx.As for interface binding)
- Modify: `internal/services/fx.go` (use fx.As to provide as interfaces)

- [ ] **Step 1: Create `internal/services/interfaces.go`**

```go
package services

import (
	"better-feature-flag/internal/models"
	"context"
)

type FeatureFlagEvaluator interface {
	EvaluateFlags(ctx context.Context, flags []models.FlagDefinition, clientCtx *models.ClientContext) (map[string]interface{}, error)
	HealthCheck(ctx context.Context, flags []models.FlagDefinition) error
}

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*models.TokenClaims, error)
}

type FlagRegistry interface {
	GetFlagsForApp(appName string) ([]models.FlagDefinition, error)
	GetAnyFlags() ([]models.FlagDefinition, error)
}
```

- [ ] **Step 2: Update `internal/middleware/auth.go`**

Change the struct field and constructor to use the interface:

```go
package middleware

import (
	"better-feature-flag/internal/models"
	"better-feature-flag/internal/services"
	"strings"

	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	tokenValidator services.TokenValidator
}

func NewAuthMiddleware(tokenValidator services.TokenValidator) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator: tokenValidator,
	}
}
```

In `OptionalJWT()`, change line 31 from `m.keycloakService.ValidateToken(...)` to `m.tokenValidator.ValidateToken(...)`.

- [ ] **Step 3: Update `internal/handlers/flags.go`**

Replace the concrete service with both interfaces. Add `?app=` query param support:

```go
package handlers

import (
	"better-feature-flag/internal/middleware"
	"better-feature-flag/internal/models"
	"better-feature-flag/internal/services"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type FlagsHandler struct {
	evaluator services.FeatureFlagEvaluator
	registry  services.FlagRegistry
	logger    *slog.Logger
}

func NewFlagsHandler(evaluator services.FeatureFlagEvaluator, registry services.FlagRegistry, logger *slog.Logger) *FlagsHandler {
	return &FlagsHandler{
		evaluator: evaluator,
		registry:  registry,
		logger:    logger,
	}
}

func (h *FlagsHandler) GetFlags(c echo.Context) error {
	ctx := c.Request().Context()
	clientCtx := middleware.GetClientContext(c)

	appName := c.QueryParam("app")
	if appName == "" {
		appName = "flutter"
	}

	flagDefs, err := h.registry.GetFlagsForApp(appName)
	if err != nil {
		h.logger.Warn("unknown app requested", slog.String("app", appName))
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Unknown application: " + appName,
		})
	}

	h.logger.Info("evaluating flags",
		slog.String("app", appName),
		slog.String("targeting_key", clientCtx.GetTargetingKey()),
		slog.String("app_version", clientCtx.AppVersion),
		slog.String("platform", clientCtx.Platform),
		slog.Bool("authenticated", clientCtx.IsAuthenticated()),
	)

	flags, err := h.evaluator.EvaluateFlags(ctx, flagDefs, clientCtx)
	if err != nil {
		h.logger.Error("failed to evaluate flags", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to evaluate feature flags",
		})
	}

	return c.JSON(http.StatusOK, models.FlagsResponse{
		Flags: flags,
	})
}
```

- [ ] **Step 4: Update `internal/handlers/health.go`**

Replace concrete service with interfaces:

```go
package handlers

import (
	"better-feature-flag/internal/models"
	"better-feature-flag/internal/services"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	evaluator services.FeatureFlagEvaluator
	registry  services.FlagRegistry
	logger    *slog.Logger
}

func NewHealthHandler(evaluator services.FeatureFlagEvaluator, registry services.FlagRegistry, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		evaluator: evaluator,
		registry:  registry,
		logger:    logger,
	}
}

func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, models.HealthResponse{
		Status: "ok",
	})
}

func (h *HealthHandler) Ready(c echo.Context) error {
	ctx := c.Request().Context()

	flags, err := h.registry.GetAnyFlags()
	if err != nil {
		h.logger.Error("readiness check failed: no flags available", slog.String("error", err.Error()))
		return c.JSON(http.StatusServiceUnavailable, models.HealthResponse{
			Status:  "unavailable",
			Message: "No flags configured",
		})
	}

	if err := h.evaluator.HealthCheck(ctx, flags); err != nil {
		h.logger.Error("readiness check failed", slog.String("error", err.Error()))
		return c.JSON(http.StatusServiceUnavailable, models.HealthResponse{
			Status:  "unavailable",
			Message: "GO Feature Flag service is not available",
		})
	}

	return c.JSON(http.StatusOK, models.HealthResponse{
		Status: "ready",
	})
}
```

- [ ] **Step 5: Update `internal/services/fx.go`**

Use `fx.As` to provide concrete types as their interfaces:

```go
package services

import (
	"go.uber.org/fx"
)

var Module = fx.Module("services",
	fx.Provide(
		fx.Annotate(
			NewFeatureFlagService,
			fx.As(new(FeatureFlagEvaluator)),
		),
		fx.Annotate(
			NewKeycloakService,
			fx.As(new(TokenValidator)),
		),
	),
)
```

Note: `FlagRegistry` will be added here in Task 3.

- [ ] **Step 6: Update `internal/handlers/fx.go`**

No changes needed — `NewFlagsHandler` and `NewHealthHandler` now accept interfaces, and FX resolves them from the `fx.As` annotations.

- [ ] **Step 7: Update `internal/middleware/fx.go`**

No changes needed — `NewAuthMiddleware` now accepts `services.TokenValidator`, and FX resolves it from the `fx.As` annotation.

- [ ] **Step 8: Update `internal/fx/fx.go`**

Update `RouteParams` to remove concrete handler/middleware types. The struct fields already match the types returned by constructors, so no changes are needed here since FX resolves by type.

- [ ] **Step 9: Verify compilation**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go build ./...`

Expected: Compilation fails because `FlagRegistry` has no implementation yet. The handlers require it but no provider exists. This is expected — Task 3 adds the implementation.

- [ ] **Step 10: Commit**

```bash
git add internal/services/interfaces.go internal/middleware/auth.go internal/handlers/flags.go internal/handlers/health.go internal/services/fx.go
git commit -m "refactor: introduce service interfaces for dependency inversion

Add FeatureFlagEvaluator, TokenValidator, FlagRegistry interfaces.
Handlers and middleware now depend on interfaces, not concrete types.
FX provides concrete implementations via fx.As annotations."
```

---

### Task 3: Config-driven flag registry

Create the `FlagRegistryService` that loads flag definitions from `config/flags.yaml`. This unblocks compilation since handlers depend on `FlagRegistry`.

**Files:**
- Create: `config/flags.yaml`
- Create: `internal/services/registry.go`
- Modify: `internal/config/config.go:17-19` (add `FlagsFile` to `AppConfig`)
- Modify: `internal/services/fx.go` (add registry provider)
- Modify: `flags/flutter.yaml` (rename `dark-mode` to `dark_mode`)
- Modify: `flags/shared.yaml` (rename all kebab-case to snake_case)

- [ ] **Step 1: Create `config/flags.yaml`**

```yaml
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

- [ ] **Step 2: Add `FlagsFile` to `AppConfig` in `internal/config/config.go`**

```go
type AppConfig struct {
	Port      string   `mapstructure:"port"`
	LogLevel  string   `mapstructure:"log_level"`
	FlagsFile string   `mapstructure:"flags_file"`
	CorsOrigins []string `mapstructure:"cors_origins"`
	RateLimit int      `mapstructure:"rate_limit"`
}
```

Add default in `Load()` after `viper.AutomaticEnv()`:

```go
	viper.SetDefault("app.flags_file", "config/flags.yaml")
	viper.SetDefault("app.rate_limit", 100)
```

- [ ] **Step 3: Create `internal/services/registry.go`**

```go
package services

import (
	"better-feature-flag/internal/config"
	"better-feature-flag/internal/models"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type FlagRegistryService struct {
	apps   map[string][]models.FlagDefinition
	logger *slog.Logger
}

type flagsFileSchema struct {
	Apps map[string]struct {
		Flags []models.FlagDefinition `yaml:"flags"`
	} `yaml:"apps"`
}

func NewFlagRegistryService(cfg *config.Config, logger *slog.Logger) (*FlagRegistryService, error) {
	filePath := cfg.App.FlagsFile
	if filePath == "" {
		filePath = "config/flags.yaml"
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read flags file %s: %w", filePath, err)
	}

	var schema flagsFileSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse flags file: %w", err)
	}

	apps := make(map[string][]models.FlagDefinition, len(schema.Apps))
	for appName, appDef := range schema.Apps {
		for i, flag := range appDef.Flags {
			if flag.Name == "" {
				return nil, fmt.Errorf("flag at index %d in app %q has no name", i, appName)
			}
			if !isValidFlagType(flag.Type) {
				return nil, fmt.Errorf("flag %q in app %q has invalid type %q", flag.Name, appName, flag.Type)
			}
		}
		apps[appName] = appDef.Flags
	}

	logger.Info("flag registry loaded",
		slog.Int("apps", len(apps)),
	)
	for appName, flags := range apps {
		logger.Info("registered app flags",
			slog.String("app", appName),
			slog.Int("flags", len(flags)),
		)
	}

	return &FlagRegistryService{
		apps:   apps,
		logger: logger,
	}, nil
}

func (r *FlagRegistryService) GetFlagsForApp(appName string) ([]models.FlagDefinition, error) {
	flags, ok := r.apps[appName]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", appName)
	}
	return flags, nil
}

func (r *FlagRegistryService) GetAnyFlags() ([]models.FlagDefinition, error) {
	for _, flags := range r.apps {
		if len(flags) > 0 {
			return flags, nil
		}
	}
	return nil, fmt.Errorf("no flags registered")
}

func isValidFlagType(t models.FlagValueType) bool {
	switch t {
	case models.FlagValueTypeBool, models.FlagValueTypeString,
		models.FlagValueTypeInt, models.FlagValueTypeFloat:
		return true
	}
	return false
}
```

- [ ] **Step 4: Update `internal/services/fx.go` — add registry**

```go
package services

import (
	"go.uber.org/fx"
)

var Module = fx.Module("services",
	fx.Provide(
		fx.Annotate(
			NewFeatureFlagService,
			fx.As(new(FeatureFlagEvaluator)),
		),
		fx.Annotate(
			NewKeycloakService,
			fx.As(new(TokenValidator)),
		),
		fx.Annotate(
			NewFlagRegistryService,
			fx.As(new(FlagRegistry)),
		),
	),
)
```

- [ ] **Step 5: Rename kebab-case flags in `flags/flutter.yaml`**

Change `dark-mode:` to `dark_mode:` (line 2).

- [ ] **Step 6: Rename kebab-case flags in `flags/shared.yaml`**

- `maintenance-mode:` → `maintenance_mode:`
- `redis-cache-enabled:` → `redis_cache_enabled:`
- `log-level:` → `log_level:`
- `rate-limit-enabled:` → `rate_limit_enabled:`

- [ ] **Step 7: Verify compilation**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go build ./...`

Expected: PASS — all interfaces now have implementations, all consumers use interfaces.

- [ ] **Step 8: Verify the app starts**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && APP_ENV=local timeout 5 go run main.go server || true`

Expected: App starts and logs "flag registry loaded" with app count. It will fail to connect to GOFF (not running) but that's fine — we just need to verify DI wiring works.

- [ ] **Step 9: Commit**

```bash
git add config/flags.yaml internal/services/registry.go internal/config/config.go internal/services/fx.go flags/flutter.yaml flags/shared.yaml
git commit -m "feat: config-driven flag registry with multi-app support

Replace hardcoded FrontendFlagDefinitions with YAML-based registry.
Flags defined in config/flags.yaml per app. Endpoint accepts ?app= param.
Rename all kebab-case flags to snake_case for consistency."
```

---

### Task 4: Observability — request ID and log level

Add request ID middleware and make log level configurable from config.

**Files:**
- Create: `internal/middleware/requestid.go`
- Modify: `internal/fx/fx.go:23-29` (ProvideLogger accepts config)
- Modify: `internal/middleware/fx.go` (register request ID middleware)
- Modify: `internal/fx/fx.go:38-50` (RouteParams adds request ID middleware)

- [ ] **Step 1: Create `internal/middleware/requestid.go`**

```go
package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/segmentio/ksuid"
)

func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = ksuid.New().String()
			}
			c.Response().Header().Set("X-Request-ID", requestID)
			c.Set("request_id", requestID)
			return next(c)
		}
	}
}
```

Note: `ksuid` is already in `go.sum` as a transitive dependency (`github.com/segmentio/ksuid v1.0.4`). Add it to `go.mod` as direct.

- [ ] **Step 2: Update `internal/fx/fx.go` — ProvideLogger accepts config**

```go
func ProvideLogger(cfg *config.Config) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.App.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)
	return logger
}
```

- [ ] **Step 3: Update `internal/middleware/fx.go`**

Add request ID middleware:

```go
var Module = fx.Module("middleware",
	fx.Provide(
		NewAuthMiddleware,
		fx.Annotate(
			CORS,
			fx.ResultTags(`name:"cors"`),
		),
		fx.Annotate(
			Logger,
			fx.ResultTags(`name:"logger"`),
		),
		fx.Annotate(
			RequestID,
			fx.ResultTags(`name:"requestid"`),
		),
	),
)
```

- [ ] **Step 4: Update `internal/fx/fx.go` — add RequestID to RouteParams and route registration**

Add to `RouteParams`:
```go
	RequestIDMiddleware echo.MiddlewareFunc `name:"requestid"`
```

Add in `RegisterRoutes` before other middleware:
```go
	p.Echo.Use(p.RequestIDMiddleware)
```

- [ ] **Step 5: Update logger middleware to include request ID**

In `internal/middleware/logger.go`, add request ID to log output:

```go
func Logger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			req := c.Request()
			res := c.Response()

			requestID, _ := c.Get("request_id").(string)

			logger.Info("request",
				slog.String("request_id", requestID),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", res.Status),
				slog.Duration("latency", time.Since(start)),
				slog.String("ip", c.RealIP()),
				slog.String("user_agent", req.UserAgent()),
			)

			return err
		}
	}
}
```

- [ ] **Step 6: Run go mod tidy (promotes ksuid to direct dependency)**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go mod tidy`

- [ ] **Step 7: Verify compilation**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go build ./...`

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/middleware/requestid.go internal/middleware/logger.go internal/middleware/fx.go internal/fx/fx.go go.mod go.sum
git commit -m "feat: add request ID middleware and configurable log level

Generate X-Request-ID per request (or propagate from header).
Include request_id in all request logs. ProvideLogger now reads
log_level from config instead of hardcoding Info."
```

---

### Task 5: Security — CORS, secrets, rate limiting, gitignore

**Files:**
- Create: `internal/middleware/ratelimit.go`
- Modify: `internal/middleware/cors.go` (config-driven origins, fix headers)
- Modify: `internal/middleware/fx.go` (CORS accepts config, add rate limiter)
- Modify: `internal/fx/fx.go` (add rate limiter to routes)
- Modify: `config/local.yaml` (remove secret, add cors_origins)
- Modify: `config/production.yaml` (add cors_origins)
- Modify: `.gitignore` (add `bin/`)

- [ ] **Step 1: Update `internal/middleware/cors.go` — config-driven**

```go
package middleware

import (
	"better-feature-flag/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func CORS(cfg *config.Config) echo.MiddlewareFunc {
	origins := cfg.App.CorsOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			"Device-ID",
			"Platform",
			"Platform-Version",
			"Device-Model",
			"Device-Architecture",
			"Device-Brand",
			"Mobile",
			"Device",
			"App-Name",
			"App-Version",
			"Package-Name",
			"Build-Number",
		},
		ExposeHeaders: []string{echo.HeaderContentLength, "X-Request-ID"},
		MaxAge:        3600,
	})
}
```

- [ ] **Step 2: Create `internal/middleware/ratelimit.go`**

```go
package middleware

import (
	"better-feature-flag/internal/config"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"better-feature-flag/internal/models"
)

type ipLimiter struct {
	tokens    float64
	lastCheck time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	rate     float64
}

func NewRateLimiter(cfg *config.Config) *RateLimiter {
	rate := float64(cfg.App.RateLimit)
	if rate <= 0 {
		rate = 100
	}
	return &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		rate:     rate,
	}
}

func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			rl.mu.Lock()
			lim, exists := rl.limiters[ip]
			if !exists {
				lim = &ipLimiter{tokens: rl.rate, lastCheck: time.Now()}
				rl.limiters[ip] = lim
			}

			now := time.Now()
			elapsed := now.Sub(lim.lastCheck).Seconds()
			lim.tokens += elapsed * rl.rate
			if lim.tokens > rl.rate {
				lim.tokens = rl.rate
			}
			lim.lastCheck = now

			if lim.tokens < 1 {
				rl.mu.Unlock()
				return c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
					Error: "Rate limit exceeded",
				})
			}

			lim.tokens--
			rl.mu.Unlock()

			return next(c)
		}
	}
}
```

- [ ] **Step 3: Update `internal/middleware/fx.go`**

```go
package middleware

import (
	"go.uber.org/fx"
)

var Module = fx.Module("middleware",
	fx.Provide(
		NewAuthMiddleware,
		NewRateLimiter,
		fx.Annotate(
			CORS,
			fx.ResultTags(`name:"cors"`),
		),
		fx.Annotate(
			Logger,
			fx.ResultTags(`name:"logger"`),
		),
		fx.Annotate(
			RequestID,
			fx.ResultTags(`name:"requestid"`),
		),
	),
)
```

- [ ] **Step 4: Update `internal/fx/fx.go` — add rate limiter to RouteParams and API group**

Add to `RouteParams`:
```go
	RateLimiter *middleware.RateLimiter
```

In `RegisterRoutes`, add rate limiter to the API group:
```go
	api := p.Echo.Group("/api/v1")
	api.Use(p.RateLimiter.Middleware())
	api.Use(p.AuthMiddleware.OptionalJWT())
	api.GET("/flags", p.FlagsHandler.GetFlags)
```

- [ ] **Step 5: Add `.env` loading to `internal/config/config.go`**

Viper's `AutomaticEnv()` reads OS environment variables but does NOT load `.env` files. Since we're removing the hardcoded secret from `local.yaml`, we need to load `.env` in the config loader. The `subosito/gotenv` package is already an indirect dependency in `go.sum`.

Add to `Load()` function, before `viper.ReadInConfig()`:

```go
	// Load .env file if present (for local development secrets)
	if data, err := os.ReadFile(".env"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if k, v, ok := strings.Cut(line, "="); ok {
				k = strings.TrimSpace(k)
				v = strings.TrimSpace(v)
				if os.Getenv(k) == "" {
					os.Setenv(k, v)
				}
			}
		}
	}
```

This is a simple `.env` parser that sets env vars only if not already set. It uses only stdlib — no new dependency needed.

- [ ] **Step 6: Update `config/local.yaml`**

Remove the hardcoded secret and the unused `jwt` section:

```yaml
app:
  version: "1.0.0"
  port: "1324"
  mode: "local"
  log_level: "debug"
  cors_origins:
    - "*"

goff:
  endpoint: "http://localhost:1031"
  environment: "development"

keycloak:
  url: "https://auth.bettercity.app"
  realm: "bettercity"
  client_id: "better-feature-flag"
  client_secret: "" # Set via KEYCLOAK_CLIENT_SECRET env var or .env file
```

Note: The `jwt` section present in the old `local.yaml` is removed — it was unused by the application.

- [ ] **Step 7: Update `config/production.yaml`**

```yaml
app:
  version: "1.0.0"
  port: "80"
  mode: "production"
  log_level: "info"
  cors_origins:
    - "https://app.bettercity.app"
    - "https://admin.bettercity.app"

goff:
  endpoint: "http://relay:1031"
  environment: "production"

keycloak:
  url: "https://auth.bettercity.app/"
  realm: "bettercity"
  client_id: "better-feature-flag"
  client_secret: "" # Overridden by KEYCLOAK_CLIENT_SECRET env var in production
```

Note: `env://` was a documentation-only convention — Viper does not parse it. The actual value comes from `KEYCLOAK_CLIENT_SECRET` env var via `AutomaticEnv()`. The new format uses an empty string with a comment, which is clearer.

- [ ] **Step 8: Update `.gitignore` — add `bin/`**

Add at the end of the file:
```
# Build output
bin/
```

- [ ] **Step 9: Remove tracked binaries**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && git rm -r --cached bin/ 2>/dev/null || true`

- [ ] **Step 10: Verify compilation**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go build ./...`

Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add internal/middleware/cors.go internal/middleware/ratelimit.go internal/middleware/fx.go internal/fx/fx.go internal/config/config.go config/local.yaml config/production.yaml .gitignore
git rm -r --cached bin/ 2>/dev/null || true
git commit -m "feat: add security hardening — CORS config, rate limiting, secrets cleanup

CORS origins now configurable per environment. Fix AllowHeaders to match
actual device headers. Add token-bucket rate limiter per IP.
Remove committed secrets from local.yaml. Remove tracked binaries."
```

---

### Task 6: Tests — flag registry

Write tests for the flag registry service using testify.

**Files:**
- Create: `internal/services/registry_test.go`
- Create: `testdata/valid_flags.yaml`
- Create: `testdata/invalid_type.yaml`

- [ ] **Step 1: Add testify dependency and tidy modules**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go get github.com/stretchr/testify@latest && go mod tidy`

- [ ] **Step 2: Create test fixtures**

Create `testdata/valid_flags.yaml`:
```yaml
apps:
  flutter:
    flags:
      - name: dark_mode
        type: bool
        default: false
      - name: app_version
        type: string
        default: "1.0.0"
  backend:
    flags:
      - name: cache_enabled
        type: bool
        default: true
```

Create `testdata/invalid_type.yaml`:
```yaml
apps:
  test:
    flags:
      - name: bad_flag
        type: invalid
        default: false
```

- [ ] **Step 3: Write `internal/services/registry_test.go`**

```go
package services_test

import (
	"better-feature-flag/internal/config"
	"better-feature-flag/internal/models"
	"better-feature-flag/internal/services"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestFlagRegistry_LoadValid(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/valid_flags.yaml"}}
	registry, err := services.NewFlagRegistryService(cfg, testLogger())
	require.NoError(t, err)

	flags, err := registry.GetFlagsForApp("flutter")
	require.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.Equal(t, "dark_mode", flags[0].Name)
	assert.Equal(t, models.FlagValueTypeBool, flags[0].Type)
}

func TestFlagRegistry_UnknownApp(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/valid_flags.yaml"}}
	registry, err := services.NewFlagRegistryService(cfg, testLogger())
	require.NoError(t, err)

	_, err = registry.GetFlagsForApp("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown app")
}

func TestFlagRegistry_InvalidType(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/invalid_type.yaml"}}
	_, err := services.NewFlagRegistryService(cfg, testLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestFlagRegistry_FileNotFound(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "nonexistent.yaml"}}
	_, err := services.NewFlagRegistryService(cfg, testLogger())
	assert.Error(t, err)
}

func TestFlagRegistry_GetAnyFlags(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/valid_flags.yaml"}}
	registry, err := services.NewFlagRegistryService(cfg, testLogger())
	require.NoError(t, err)

	flags, err := registry.GetAnyFlags()
	require.NoError(t, err)
	assert.NotEmpty(t, flags)
}

func TestFlagRegistry_MultipleApps(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/valid_flags.yaml"}}
	registry, err := services.NewFlagRegistryService(cfg, testLogger())
	require.NoError(t, err)

	flutter, err := registry.GetFlagsForApp("flutter")
	require.NoError(t, err)
	assert.Len(t, flutter, 2)

	backend, err := registry.GetFlagsForApp("backend")
	require.NoError(t, err)
	assert.Len(t, backend, 1)
	assert.Equal(t, "cache_enabled", backend[0].Name)
}
```

- [ ] **Step 4: Run tests**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go test ./internal/services/ -v -run TestFlagRegistry`

Expected: All 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add testdata/ internal/services/registry_test.go go.mod go.sum
git commit -m "test: add flag registry tests with testify

Cover: valid loading, unknown app, invalid type, missing file,
GetAnyFlags, multi-app support."
```

---

### Task 7: Tests — middleware (auth + request ID)

**Files:**
- Create: `internal/middleware/auth_test.go`
- Create: `internal/middleware/requestid_test.go`

- [ ] **Step 1: Write `internal/middleware/auth_test.go`**

```go
package middleware_test

import (
	"better-feature-flag/internal/middleware"
	"better-feature-flag/internal/models"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTokenValidator struct {
	claims *models.TokenClaims
	err    error
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, token string) (*models.TokenClaims, error) {
	return m.claims, m.err
}

func TestOptionalJWT_WithValidToken(t *testing.T) {
	validator := &mockTokenValidator{
		claims: &models.TokenClaims{Sub: "user-123", Email: "test@test.com", Username: "testuser"},
	}
	auth := middleware.NewAuthMiddleware(validator)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Device-ID", "device-abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedCtx *models.ClientContext
	handler := auth.OptionalJWT()(func(c echo.Context) error {
		capturedCtx = middleware.GetClientContext(c)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, "user-123", capturedCtx.UserID)
	assert.Equal(t, "test@test.com", capturedCtx.Email)
	assert.Equal(t, "device-abc", capturedCtx.DeviceID)
}

func TestOptionalJWT_WithoutToken(t *testing.T) {
	validator := &mockTokenValidator{}
	auth := middleware.NewAuthMiddleware(validator)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Device-ID", "device-abc")
	req.Header.Set("Platform", "android")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedCtx *models.ClientContext
	handler := auth.OptionalJWT()(func(c echo.Context) error {
		capturedCtx = middleware.GetClientContext(c)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Empty(t, capturedCtx.UserID)
	assert.Equal(t, "device-abc", capturedCtx.DeviceID)
	assert.Equal(t, "android", capturedCtx.Platform)
}

func TestOptionalJWT_WithInvalidToken(t *testing.T) {
	validator := &mockTokenValidator{err: fmt.Errorf("token expired")}
	auth := middleware.NewAuthMiddleware(validator)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.Header.Set("Device-ID", "device-abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedCtx *models.ClientContext
	handler := auth.OptionalJWT()(func(c echo.Context) error {
		capturedCtx = middleware.GetClientContext(c)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	// Should still proceed without user info
	assert.Empty(t, capturedCtx.UserID)
	assert.Equal(t, "device-abc", capturedCtx.DeviceID)
}

func TestGetClientContext_NoContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ctx := middleware.GetClientContext(c)
	assert.NotNil(t, ctx)
	assert.Empty(t, ctx.UserID)
}
```

- [ ] **Step 2: Write `internal/middleware/requestid_test.go`**

```go
package middleware_test

import (
	"better-feature-flag/internal/middleware"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesNew(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedID string
	handler := middleware.RequestID()(func(c echo.Context) error {
		capturedID, _ = c.Get("request_id").(string)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.NotEmpty(t, capturedID)
	assert.Equal(t, capturedID, rec.Header().Get("X-Request-ID"))
}

func TestRequestID_PropagatesExisting(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "existing-id-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedID string
	handler := middleware.RequestID()(func(c echo.Context) error {
		capturedID, _ = c.Get("request_id").(string)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, "existing-id-123", capturedID)
	assert.Equal(t, "existing-id-123", rec.Header().Get("X-Request-ID"))
}
```

- [ ] **Step 3: Run tests**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go test ./internal/middleware/ -v`

Expected: All 6 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/middleware/auth_test.go internal/middleware/requestid_test.go
git commit -m "test: add middleware tests for auth and request ID

Cover: valid token, no token, invalid token, device headers,
GetClientContext fallback, request ID generation and propagation."
```

---

### Task 8: Tests — handlers (flags + health)

**Files:**
- Create: `internal/handlers/flags_test.go`
- Create: `internal/handlers/health_test.go`

- [ ] **Step 1: Write `internal/handlers/flags_test.go`**

```go
package handlers_test

import (
	"better-feature-flag/internal/handlers"
	"better-feature-flag/internal/models"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEvaluator struct {
	result map[string]interface{}
	err    error
}

func (m *mockEvaluator) EvaluateFlags(ctx context.Context, flags []models.FlagDefinition, clientCtx *models.ClientContext) (map[string]interface{}, error) {
	return m.result, m.err
}

func (m *mockEvaluator) HealthCheck(ctx context.Context, flags []models.FlagDefinition) error {
	return m.err
}

type mockRegistry struct {
	apps map[string][]models.FlagDefinition
}

func (m *mockRegistry) GetFlagsForApp(appName string) ([]models.FlagDefinition, error) {
	flags, ok := m.apps[appName]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", appName)
	}
	return flags, nil
}

func (m *mockRegistry) GetAnyFlags() ([]models.FlagDefinition, error) {
	for _, flags := range m.apps {
		return flags, nil
	}
	return nil, fmt.Errorf("no flags")
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestGetFlags_Success(t *testing.T) {
	eval := &mockEvaluator{result: map[string]interface{}{"dark_mode": true}}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "dark_mode", Type: models.FlagValueTypeBool, Default: false}},
	}}
	h := handlers.NewFlagsHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags?app=flutter", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("client_context", &models.ClientContext{DeviceID: "dev-1"})

	err := h.GetFlags(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "dark_mode")
}

func TestGetFlags_DefaultApp(t *testing.T) {
	eval := &mockEvaluator{result: map[string]interface{}{"dark_mode": false}}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "dark_mode", Type: models.FlagValueTypeBool, Default: false}},
	}}
	h := handlers.NewFlagsHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("client_context", &models.ClientContext{})

	err := h.GetFlags(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetFlags_UnknownApp(t *testing.T) {
	eval := &mockEvaluator{}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{}}
	h := handlers.NewFlagsHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags?app=unknown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("client_context", &models.ClientContext{})

	err := h.GetFlags(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetFlags_EvaluationError(t *testing.T) {
	eval := &mockEvaluator{err: fmt.Errorf("goff down")}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "dark_mode", Type: models.FlagValueTypeBool, Default: false}},
	}}
	h := handlers.NewFlagsHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags?app=flutter", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("client_context", &models.ClientContext{})

	err := h.GetFlags(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
```

- [ ] **Step 2: Write `internal/handlers/health_test.go`**

```go
package handlers_test

import (
	"better-feature-flag/internal/handlers"
	"better-feature-flag/internal/models"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth_AlwaysOK(t *testing.T) {
	eval := &mockEvaluator{}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "test", Type: models.FlagValueTypeBool}},
	}}
	h := handlers.NewHealthHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Health(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}

func TestReady_Success(t *testing.T) {
	eval := &mockEvaluator{}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "test", Type: models.FlagValueTypeBool}},
	}}
	h := handlers.NewHealthHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Ready(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ready")
}

func TestReady_GOFFUnavailable(t *testing.T) {
	eval := &mockEvaluator{err: fmt.Errorf("connection refused")}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "test", Type: models.FlagValueTypeBool}},
	}}
	h := handlers.NewHealthHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Ready(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
```

- [ ] **Step 3: Run all tests**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go test ./... -v -race`

Expected: All tests PASS across all packages.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/flags_test.go internal/handlers/health_test.go
git commit -m "test: add handler tests for flags and health endpoints

Cover: successful evaluation, default app, unknown app, evaluation error,
liveness probe, readiness success, GOFF unavailable."
```

---

### Task 9: CI pipeline — add test gate

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Update `.github/workflows/ci.yml`**

Insert `test` job before existing jobs and add `needs: test` to both:

```yaml
name: CI

on:
  push:
    branches: [main]
    tags: ["v*"]
  workflow_dispatch:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Run tests
        run: go test ./... -race -coverprofile=coverage.out

      - name: Run vet
        run: go vet ./...

  relay:
    needs: test
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - name: Checkout app repo
        uses: actions/checkout@v4

      - name: Checkout git-actions
        uses: actions/checkout@v4
        with:
          repository: develogo/git-actions
          token: ${{ secrets.STACKS_REPO_TOKEN }}
          path: git-actions

      - name: Build, push and open PR in stacks
        uses: ./git-actions/actions/build_push_and_update_stacks
        with:
          image_name: bettercity-flags
          image_repo: develogo/bettercity-flags
          stacks_repo: develogo/stacks
          stacks_compose_path: stacks/bettercity-feature-flag/docker-compose.yml
          stacks_repo_token: ${{ secrets.STACKS_REPO_TOKEN }}
          dockerfile_path: Dockerfile
          context: .
          platforms: linux/amd64

  api:
    needs: test
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - name: Checkout app repo
        uses: actions/checkout@v4

      - name: Checkout git-actions
        uses: actions/checkout@v4
        with:
          repository: develogo/git-actions
          token: ${{ secrets.STACKS_REPO_TOKEN }}
          path: git-actions

      - name: Build, push and open PR in stacks
        uses: ./git-actions/actions/build_push_and_update_stacks
        with:
          image_name: better-feature-flag
          image_repo: develogo/better-feature-flag
          stacks_repo: develogo/stacks
          stacks_compose_path: stacks/bettercity-feature-flag/docker-compose.yml
          stacks_repo_token: ${{ secrets.STACKS_REPO_TOKEN }}
          dockerfile_path: Dockerfile.api
          context: .
          platforms: linux/amd64
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add test gate before image builds

New test job runs go test and go vet. Both relay and api jobs
now depend on test passing before building images."
```

---

### Task 10: Flag files reorganization and Docker

Move flag files, update GOFF config, update Docker Compose and Makefile.

**Files:**
- Move: `flags/flutter.yaml` → `flags/apps/flutter.yaml`
- Modify: `goff-proxy.yaml` (update path)
- Modify: `docker-compose.yml` (full rewrite per spec)
- Modify: `Makefile` (update help text, add test target)

- [ ] **Step 1: Move flutter.yaml to apps subdirectory**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && mkdir -p flags/apps && git mv flags/flutter.yaml flags/apps/flutter.yaml`

- [ ] **Step 2: Update `goff-proxy.yaml`**

```yaml
# GO Feature Flag Relay Proxy
pollingInterval: 1000

retrievers:
  # Flutter flags
  - kind: file
    path: /goff/flags/apps/flutter.yaml

  # Shared flags
  - kind: file
    path: /goff/flags/shared.yaml
```

- [ ] **Step 3: Update `docker-compose.yml`**

```yaml
services:
  relay:
    image: gofeatureflag/go-feature-flag:latest
    ports:
      - "1031:1031"
    volumes:
      - ./flags:/goff/flags
      - ./goff-proxy.yaml:/goff/goff-proxy.yaml
    restart: unless-stopped

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
    restart: unless-stopped

networks:
  default:
    name: bettercity_local
```

- [ ] **Step 4: Update `Makefile`**

```makefile
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
```

- [ ] **Step 5: Verify compilation and tests**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go build ./... && go test ./... -race`

Expected: Build and all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add flags/apps/flutter.yaml goff-proxy.yaml docker-compose.yml Makefile
git rm flags/flutter.yaml 2>/dev/null || true
git commit -m "infra: reorganize flag files, update Docker and Makefile

Move flutter.yaml to flags/apps/. Update goff-proxy.yaml paths.
Docker Compose now includes API server and creates network automatically.
Add make test target."
```

---

### Task 11: Final verification and cleanup

- [ ] **Step 1: Run full test suite**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go test ./... -race -coverprofile=coverage.out -v`

Expected: All tests PASS.

- [ ] **Step 2: Run go vet**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go vet ./...`

Expected: No issues.

- [ ] **Step 3: Verify build**

Run: `cd /c/Users/lippe/Documents/GitHub/bettercity/better-feature-flag && go build -o bin/server .`

Expected: Binary created successfully.

- [ ] **Step 4: Update CLAUDE.md**

Update the CLAUDE.md to reflect the new architecture (interfaces, flag registry, test commands, new config fields). Key additions:
- `make test` / `go test ./... -race` for running tests
- Flag registry in `config/flags.yaml`
- New config fields: `cors_origins`, `rate_limit`, `flags_file`, `log_level`
- Interface-based architecture
- `?app=` query parameter on `/api/v1/flags`

- [ ] **Step 5: Final commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md for new architecture"
```

---

### Deferred Items

The following spec requirement is intentionally deferred from this plan:

- **GOFF startup flag mismatch validation** (Spec Section 2: "Flags defined in the registry but absent in GOFF emit a warning log"). This would require calling the GOFF relay at startup to list its flags and cross-checking. Since the relay may not be available during API server startup (especially in Docker where startup order isn't guaranteed), this validation is better implemented as a background check after the server is running, or as part of the readiness probe. Deferred to a future iteration.
