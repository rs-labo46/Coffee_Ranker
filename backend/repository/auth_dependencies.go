package repository

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"golang.org/x/crypto/bcrypt"
)

type passwordHasher struct{}

func NewPasswordHasher() *passwordHasher {
	return &passwordHasher{}
}

func (h *passwordHasher) Hash(ctx context.Context, password string) (string, error) {
	_ = ctx

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (h *passwordHasher) Compare(ctx context.Context, password string, passwordHash string) error {
	_ = ctx

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return entity.ErrInvalidCredentials
	}

	return nil
}

type tokenManager struct {
	secret    []byte
	accessTTL time.Duration
	users     IUserRepository
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type accessTokenClaims struct {
	UserID       uint64          `json:"user_id"`
	Role         entity.UserRole `json:"role"`
	TokenVersion int             `json:"token_version"`
	ExpiresAt    int64           `json:"exp"`
	IssuedAt     int64           `json:"iat"`
}

func NewTokenManager(secret string, accessTTL time.Duration, users IUserRepository) (*tokenManager, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 bytes")
	}
	if accessTTL <= 0 {
		return nil, errors.New("access token ttl must be positive")
	}
	if users == nil {
		return nil, errors.New("user repository is required")
	}

	return &tokenManager{
		secret:    []byte(secret),
		accessTTL: accessTTL,
		users:     users,
	}, nil
}

func (m *tokenManager) NewRefreshToken(ctx context.Context) (plain string, hash string, err error) {
	_ = ctx

	plain, err = randomToken("rt_", 32)
	if err != nil {
		return "", "", err
	}

	return plain, m.hashToken(plain), nil
}

func (m *tokenManager) NewFamilyID(ctx context.Context) (string, error) {
	_ = ctx
	return randomToken("rtf_", 24)
}

func (m *tokenManager) HashRefreshToken(ctx context.Context, token string) (string, error) {
	_ = ctx

	token = strings.TrimSpace(token)
	if token == "" {
		return "", entity.ErrInvalidToken
	}

	return m.hashToken(token), nil
}

func (m *tokenManager) IssueAccessToken(ctx context.Context, user *model.User, now time.Time) (string, error) {
	_ = ctx

	if user == nil || user.ID == 0 {
		return "", entity.ErrUnauthorized
	}
	if user.Role != entity.UserRoleUser && user.Role != entity.UserRoleAdmin {
		return "", entity.ErrUnauthorized
	}
	if user.Status != entity.UserStatusActive {
		return "", entity.ErrUnauthorized
	}

	claims := accessTokenClaims{
		UserID:       user.ID,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		ExpiresAt:    now.Add(m.accessTTL).Unix(),
		IssuedAt:     now.Unix(),
	}

	return m.signJWT(claims)
}

func (m *tokenManager) ValidateAccessToken(ctx context.Context, rawToken string) (uint64, entity.UserRole, int, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return 0, "", 0, entity.ErrInvalidToken
	}

	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return 0, "", 0, entity.ErrInvalidToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, "", 0, entity.ErrInvalidToken
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return 0, "", 0, entity.ErrInvalidToken
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return 0, "", 0, entity.ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]

	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return 0, "", 0, entity.ErrInvalidToken
	}

	expectedSig := m.signature(unsigned)
	if !hmac.Equal(gotSig, expectedSig) {
		return 0, "", 0, entity.ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, "", 0, entity.ErrInvalidToken
	}

	var claims accessTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, "", 0, entity.ErrInvalidToken
	}

	if claims.UserID == 0 || claims.ExpiresAt <= time.Now().Unix() {
		return 0, "", 0, entity.ErrUnauthorized
	}
	if claims.Role != entity.UserRoleUser && claims.Role != entity.UserRoleAdmin {
		return 0, "", 0, entity.ErrUnauthorized
	}

	user, err := m.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return 0, "", 0, entity.ErrUnauthorized
	}

	switch user.Status {
	case entity.UserStatusActive:
	case entity.UserStatusSuspended:
		return 0, "", 0, entity.ErrUserSuspended
	case entity.UserStatusDeleted:
		return 0, "", 0, entity.ErrUserDeleted
	default:
		return 0, "", 0, entity.ErrUnauthorized
	}

	if user.Role != claims.Role || user.TokenVersion != claims.TokenVersion {
		return 0, "", 0, entity.ErrUnauthorized
	}

	return user.ID, user.Role, user.TokenVersion, nil
}

func (m *tokenManager) signJWT(claims accessTokenClaims) (string, error) {
	headerJSON, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := header + "." + payload
	signature := base64.RawURLEncoding.EncodeToString(m.signature(unsigned))

	return unsigned + "." + signature, nil
}

func (m *tokenManager) signature(value string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func (m *tokenManager) hashToken(token string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

type guestKeyManager struct{}

func NewGuestKeyManager() *guestKeyManager {
	return &guestKeyManager{}
}

func (m *guestKeyManager) NewGuestSessionKey(ctx context.Context) (plain string, hash string, err error) {
	_ = ctx

	plain, err = randomToken("gs_", 32)
	if err != nil {
		return "", "", err
	}

	return plain, sha256Hex(plain), nil
}

func (m *guestKeyManager) HashGuestSessionKey(ctx context.Context, key string) (string, error) {
	_ = ctx

	key = strings.TrimSpace(key)
	if key == "" {
		return "", entity.ErrInvalidInput
	}

	return sha256Hex(key), nil
}

func randomToken(prefix string, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return prefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
