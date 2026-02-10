package controlplane

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

const (
	jwtExpiryDuration = 24 * time.Hour
	bcryptCost        = 12
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// RegisterRequest represents user registration input
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	TenantName  string `json:"tenant_name"`
	TenantSlug  string `json:"tenant_slug"`
}

// LoginRequest represents user login input
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the login/register response
type AuthResponse struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      UserInfo    `json:"user"`
	Tenant    TenantInfo  `json:"tenant"`
}

// UserInfo represents user data in auth response
type UserInfo struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// TenantInfo represents tenant data in auth response
type TenantInfo struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	PrimaryRegion string `json:"primary_region"`
}

// getJWTSecret returns the JWT signing secret from config
func (a *App) getJWTSecret() []byte {
	// In production, this should come from environment/config
	// For now, generate a deterministic secret from admin key or use a default
	if a.cfg.AdminKey != "" {
		// Use first 32 bytes of admin key as JWT secret
		secret := []byte(a.cfg.AdminKey)
		if len(secret) >= 32 {
			return secret[:32]
		}
		// Pad if needed
		padded := make([]byte, 32)
		copy(padded, secret)
		return padded
	}
	// Fallback secret (not for production!)
	return []byte("n-kudo-development-jwt-secret-key")
}

// generateToken creates a JWT token for a user
func (a *App) generateToken(user store.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(jwtExpiryDuration)
	
	claims := JWTClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "n-kudo",
			Subject:   user.ID,
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.getJWTSecret())
	if err != nil {
		return "", time.Time{}, err
	}
	
	return tokenString, expiresAt, nil
}

// validateToken parses and validates a JWT token
func (a *App) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.getJWTSecret(), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token claims")
}

// hashPassword creates a bcrypt hash of the password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// checkPassword verifies a password against a hash
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// handleRegister handles user registration with tenant creation
func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	// Validate input
	if req.Email == "" || req.Password == "" || req.DisplayName == "" || req.TenantName == "" {
		writeError(w, http.StatusBadRequest, "email, password, display_name, and tenant_name are required")
		return
	}
	
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	
	// Check if email already exists
	exists, err := a.repo.EmailExists(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if exists {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	
	// Generate tenant slug if not provided
	tenantSlug := req.TenantSlug
	if tenantSlug == "" {
		tenantSlug = slugify(req.TenantName)
	}
	
	// Start transaction-like flow (we'll do it in sequence for simplicity)
	ctx := r.Context()
	
	// Create tenant first
	tenant := store.Tenant{
		ID:            uuid.New().String(),
		Slug:          tenantSlug,
		Name:          req.TenantName,
		PrimaryRegion: "eu-central-1",
		RetentionDays: 30,
	}
	
	createdTenant, err := a.repo.CreateTenant(ctx, tenant)
	if err != nil {
		if err == store.ErrConflict {
			writeError(w, http.StatusConflict, "tenant slug already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}
	
	// Hash password
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}
	
	// Create user as tenant owner
	user := store.User{
		ID:           uuid.New().String(),
		TenantID:     createdTenant.ID,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		Role:         "OWNER",
		PasswordHash: passwordHash,
		IsActive:     true,
		EmailVerified: false, // TODO: send verification email
	}
	
	createdUser, err := a.repo.CreateUser(ctx, user)
	if err != nil {
		// TODO: rollback tenant creation
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	
	// Generate JWT token
	tokenString, expiresAt, err := a.generateToken(createdUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	
	// Write audit event
	_ = a.writeAudit(ctx, createdTenant.ID, "", "SYSTEM", "", "user.register", "user", createdUser.ID, requestID(r), sourceIP(r), nil)
	
	// Return response
	resp := AuthResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:          createdUser.ID,
			Email:       createdUser.Email,
			DisplayName: createdUser.DisplayName,
			Role:        createdUser.Role,
		},
		Tenant: TenantInfo{
			ID:            createdTenant.ID,
			Slug:          createdTenant.Slug,
			Name:          createdTenant.Name,
			PrimaryRegion: createdTenant.PrimaryRegion,
		},
	}
	
	writeJSON(w, http.StatusCreated, resp)
}

// handleLogin handles user authentication
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	// Validate input
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	
	// Find user by email
	user, err := a.repo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	
	// Check if user is active
	if !user.IsActive {
		writeError(w, http.StatusUnauthorized, "account is disabled")
		return
	}
	
	// Verify password
	if !checkPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	
	// Update last login
	_ = a.repo.UpdateUserLastLogin(r.Context(), user.TenantID, user.ID)
	
	// Get tenant info
	tenants, err := a.repo.ListTenants(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get tenant info")
		return
	}
	
	var tenantInfo TenantInfo
	for _, t := range tenants {
		if t.ID == user.TenantID {
			tenantInfo = TenantInfo{
				ID:            t.ID,
				Slug:          t.Slug,
				Name:          t.Name,
				PrimaryRegion: t.PrimaryRegion,
			}
			break
		}
	}
	
	// Generate JWT token
	tokenString, expiresAt, err := a.generateToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	
	// Write audit event
	_ = a.writeAudit(r.Context(), user.TenantID, "", "USER", user.ID, "user.login", "user", user.ID, requestID(r), sourceIP(r), nil)
	
	// Return response
	resp := AuthResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			LastLoginAt: user.LastLoginAt,
		},
		Tenant: tenantInfo,
	}
	
	writeJSON(w, http.StatusOK, resp)
}

// handleGetMe returns the current user's profile
func (a *App) handleGetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	
	user, err := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	
	writeJSON(w, http.StatusOK, UserInfo{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		LastLoginAt: user.LastLoginAt,
	})
}

// authMiddleware validates JWT tokens and injects user context
func (a *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}
		
		// Extract Bearer token
		var tokenString string
		if _, err := fmt.Sscanf(authHeader, "Bearer %s", &tokenString); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}
		
		// Validate token
		claims, err := a.validateToken(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		
		// Inject claims into context
		ctx := context.WithValue(r.Context(), "user", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// tenantScopedAuth middleware ensures user can only access their own tenant
func (a *App) tenantScopedAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value("user").(*JWTClaims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		
		// Check if URL tenant_id matches user's tenant
		urlTenantID := r.PathValue("tenantID")
		if urlTenantID != "" && urlTenantID != claims.TenantID {
			writeError(w, http.StatusForbidden, "access denied for this tenant")
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Helper to generate random strings
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

// slugify creates a URL-friendly slug from a string
func slugify(s string) string {
	var result []rune
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result = append(result, r)
		case r >= 'A' && r <= 'Z':
			result = append(result, r+('a'-'A'))
		case r >= '0' && r <= '9':
			result = append(result, r)
		case r == ' ' || r == '-' || r == '_':
			if len(result) == 0 || result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	// Trim trailing dash
	if len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	if len(result) == 0 {
		return "untitled"
	}
	return string(result)
}

// ============================================
// Email Verification
// ============================================

// handleResendVerification resends the verification email
func (a *App) handleResendVerification(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Get user info
	user, err := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Check if already verified
	if user.EmailVerified {
		writeError(w, http.StatusBadRequest, "email already verified")
		return
	}

	// Generate verification token
	token, err := generateRandomString(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Hash token for storage
	tokenHash := hashToken(token)

	// Store token (expires in 24 hours)
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := a.repo.CreateEmailVerificationToken(r.Context(), user.ID, user.TenantID, tokenHash, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create verification token")
		return
	}

	// Send email
	if a.emailService != nil && a.emailService.IsEnabled() {
		if err := a.emailService.SendVerificationEmail(user.Email, token); err != nil {
			// Log error but don't fail the request
			log.Printf("Failed to send verification email: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Verification email sent",
	})
}

// handleVerifyEmail verifies the email address from a token
func (a *App) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	// Hash the provided token
	tokenHash := hashToken(token)

	// Verify token
	userID, tenantID, err := a.repo.VerifyEmailToken(r.Context(), tokenHash)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusBadRequest, "invalid or expired token")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify token")
		return
	}

	// Mark email as verified
	if err := a.repo.MarkEmailVerified(r.Context(), tenantID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark email as verified")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

// hashToken creates a simple hash of a token for storage
func hashToken(token string) string {
	// Use bcrypt-like approach with SHA256 for tokens
	// In production, consider using a proper KDF
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}
