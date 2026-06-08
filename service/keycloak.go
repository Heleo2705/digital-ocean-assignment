package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type KeycloakClaims struct {
	jwt.RegisteredClaims
	Subject           string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

func (c *KeycloakClaims) UserID() string {
	return c.Subject
}

func (c *KeycloakClaims) Username() string {
	if c.PreferredUsername != "" {
		return c.PreferredUsername
	}
	return c.Email
}

func (c *KeycloakClaims) Roles(clientID string) []string {
	if clientID != "" {
		if resource, ok := c.ResourceAccess[clientID]; ok {
			return resource.Roles
		}
	}
	return c.RealmAccess.Roles
}

type KeycloakService struct {
	Issuer     string
	ClientID   string
	JwksURL    string
	HTTPClient *http.Client

	mu          sync.RWMutex
	keyCache    map[string]*rsa.PublicKey
	cacheExpiry time.Time
}

func NewKeycloakService(issuer, clientID string) *KeycloakService {
	issuer = strings.TrimSpace(issuer)
	if !strings.HasPrefix(issuer, "http") {
		issuer = "https://" + issuer
	}
	issuer = strings.TrimRight(issuer, "/")

	return &KeycloakService{
		Issuer:     issuer,
		ClientID:   clientID,
		JwksURL:    issuer + "/protocol/openid-connect/certs",
		HTTPClient: http.DefaultClient,
		keyCache:   make(map[string]*rsa.PublicKey),
	}
}

func (s *KeycloakService) ValidateAccessToken(ctx context.Context, bearerToken string) (*KeycloakClaims, error) {
	if bearerToken == "" {
		return nil, errors.New("missing access token")
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(bearerToken, "Bearer"))
	if tokenString == "" {
		return nil, errors.New("invalid bearer token")
	}

	claims := &KeycloakClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("missing kid in token header")
		}

		return s.getKey(ctx, kid)
	}, jwt.WithIssuer(s.Issuer), jwt.WithAudience(s.ClientID))
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid access token")
	}

	return claims, nil
}

func (s *KeycloakService) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	s.mu.RLock()
	if key, ok := s.keyCache[kid]; ok && time.Now().Before(s.cacheExpiry) {
		s.mu.RUnlock()
		return key, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if key, ok := s.keyCache[kid]; ok && time.Now().Before(s.cacheExpiry) {
		return key, nil
	}

	jwks, err := s.fetchJWKS(ctx)
	if err != nil {
		return nil, err
	}

	pub, err := jwks.publicKeyFor(kid)
	if err != nil {
		return nil, err
	}

	s.keyCache[kid] = pub
	s.cacheExpiry = time.Now().Add(15 * time.Minute)
	return pub, nil
}

type jwks struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (j *jwks) publicKeyFor(kid string) (*rsa.PublicKey, error) {
	for _, key := range j.Keys {
		if key.Kid == kid {
			return key.rsaPublicKey()
		}
	}
	return nil, fmt.Errorf("jwks key not found for kid %s", kid)
}

func (k *jwkKey) rsaPublicKey() (*rsa.PublicKey, error) {
	if k.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key type %s", k.Kty)
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("invalid rsa modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("invalid rsa exponent: %w", err)
	}

	exponent := int(big.NewInt(0).SetBytes(eBytes).Int64())
	if exponent == 0 {
		return nil, errors.New("invalid rsa exponent")
	}

	return &rsa.PublicKey{
		N: big.NewInt(0).SetBytes(nBytes),
		E: exponent,
	}, nil
}

func (s *KeycloakService) fetchJWKS(ctx context.Context) (*jwks, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.JwksURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch jwks: %s", resp.Status)
	}

	var jwksResp jwks
	if err := json.NewDecoder(resp.Body).Decode(&jwksResp); err != nil {
		return nil, err
	}
	return &jwksResp, nil
}
