package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/golang-jwt/jwt/v5"
)

type KeycloakService struct {
	client       *gocloak.GoCloak
	realm        string
	clientID     string
	clientSecret string
	logger       *slog.Logger
}

type TokenClaims struct {
	Sub      string
	Email    string
	Username string
	Active   bool
}

type JWTClaims struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	jwt.RegisteredClaims
}

func NewKeycloakService(keycloakURL, realm, clientID, clientSecret string, logger *slog.Logger) *KeycloakService {
	client := gocloak.NewClient(keycloakURL)

	return &KeycloakService{
		client:       client,
		realm:        realm,
		clientID:     clientID,
		clientSecret: clientSecret,
		logger:       logger,
	}
}

// ValidateToken valida o token via introspection endpoint do Keycloak e extrai claims
func (s *KeycloakService) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	// Realiza introspection do token
	rptResult, err := s.client.RetrospectToken(ctx, token, s.clientID, s.clientSecret, s.realm)
	if err != nil {
		s.logger.Debug("token introspection failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to introspect token: %w", err)
	}

	// Verifica se o token está ativo
	if rptResult.Active == nil || !*rptResult.Active {
		s.logger.Debug("token is not active")
		return nil, fmt.Errorf("token is not active")
	}

	// Extrai claims decodificando o JWT (sem validar assinatura, apenas extraindo dados)
	claims, err := s.extractClaimsFromJWT(token)
	if err != nil {
		s.logger.Debug("failed to extract claims from JWT", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	claims.Active = true

	s.logger.Debug("token validated successfully",
		slog.String("sub", claims.Sub),
		slog.String("username", claims.Username),
	)

	return claims, nil
}

// extractClaimsFromJWT decodifica o JWT sem validar a assinatura (apenas para extrair claims)
func (s *KeycloakService) extractClaimsFromJWT(token string) (*TokenClaims, error) {
	// Separa as partes do JWT
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decodifica o payload (segunda parte)
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedToken, _, err := parser.ParseUnverified(token, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	jwtClaims, ok := parsedToken.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("failed to parse JWT claims")
	}

	return &TokenClaims{
		Sub:      jwtClaims.Sub,
		Email:    jwtClaims.Email,
		Username: jwtClaims.PreferredUsername,
	}, nil
}
