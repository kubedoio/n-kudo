package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// CreateProjectRequest represents a request to create a new project
type CreateProjectRequest struct {
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	PrimaryRegion string `json:"primary_region"`
}

// ProjectResponse represents a project in the response
type ProjectResponse struct {
	ID            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	PrimaryRegion string `json:"primary_region"`
	Role          string `json:"role"` // User's role in this project
	CreatedAt     string `json:"created_at"`
}

// handleListMyProjects returns all projects the authenticated user has access to
// For now, users can only access their own tenant (single tenant per user)
func (a *App) handleListMyProjects(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Get user's tenant info
	user, err := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user info")
		return
	}

	// Get tenant details
	tenant, err := a.repo.GetTenantByID(r.Context(), user.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get project info")
		return
	}

	project := ProjectResponse{
		ID:            tenant.ID,
		Slug:          tenant.Slug,
		Name:          tenant.Name,
		PrimaryRegion: tenant.PrimaryRegion,
		Role:          user.Role,
		CreatedAt:     tenant.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"projects": []ProjectResponse{project},
	})
}

// handleGetMyProject returns the user's current project
func (a *App) handleGetMyProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Get user's tenant info
	user, err := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user info")
		return
	}

	// Get tenant details
	tenant, err := a.repo.GetTenantByID(r.Context(), user.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get project info")
		return
	}

	project := ProjectResponse{
		ID:            tenant.ID,
		Slug:          tenant.Slug,
		Name:          tenant.Name,
		PrimaryRegion: tenant.PrimaryRegion,
		Role:          user.Role,
		CreatedAt:     tenant.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	writeJSON(w, http.StatusOK, project)
}

// handleSwitchProject switches the user's current project and returns a new JWT token
func (a *App) handleSwitchProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	projectID := r.PathValue("projectID")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project ID is required")
		return
	}

	// Get the current user's email
	currentUser, err := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get current user")
		return
	}

	// Look up user by email in the target project
	targetUser, err := a.repo.GetUserByEmailAndTenant(r.Context(), currentUser.Email, projectID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusForbidden, "you don't have access to this project")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to lookup user")
		return
	}

	// Generate new token for the target project
	tokenString, expiresAt, err := a.generateToken(targetUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Get tenant info
	tenant, err := a.repo.GetTenantByID(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get project info")
		return
	}

	resp := AuthResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:          targetUser.ID,
			Email:       targetUser.Email,
			DisplayName: targetUser.DisplayName,
			Role:        targetUser.Role,
			LastLoginAt: targetUser.LastLoginAt,
		},
		Tenant: TenantInfo{
			ID:            tenant.ID,
			Slug:          tenant.Slug,
			Name:          tenant.Name,
			PrimaryRegion: tenant.PrimaryRegion,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleCreateProject allows users to create a new project
// This creates a new tenant and makes the user the owner
func (a *App) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Generate slug if not provided
	tenantSlug := req.Slug
	if tenantSlug == "" {
		tenantSlug = slugify(req.Name)
	}

	// Set default region
	primaryRegion := req.PrimaryRegion
	if primaryRegion == "" {
		primaryRegion = "eu-central-1"
	}

	ctx := r.Context()

	// Create new tenant/project
	tenant := store.Tenant{
		ID:            uuid.New().String(),
		Slug:          tenantSlug,
		Name:          req.Name,
		PrimaryRegion: primaryRegion,
		RetentionDays: 30,
	}

	createdTenant, err := a.repo.CreateTenant(ctx, tenant)
	if err != nil {
		if err == store.ErrConflict {
			writeError(w, http.StatusConflict, "project slug already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	// Create user membership in the new project
	// Note: For now, we don't support multiple projects per user in the data model
	// This would require a separate user_projects join table
	// For now, we'll just return an error indicating this limitation
	// or we could create the user in the new project and they lose access to the old one

	// Actually, let me create a new user entry for this tenant
	// Get current user info
	currentUser, err := a.repo.GetUserByID(ctx, claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user info")
		return
	}

	// Create new user in the new tenant
	newUser := store.User{
		ID:            uuid.New().String(),
		TenantID:      createdTenant.ID,
		Email:         currentUser.Email,
		DisplayName:   currentUser.DisplayName,
		Role:          "OWNER",
		PasswordHash:  currentUser.PasswordHash, // Same password
		IsActive:      true,
		EmailVerified: currentUser.EmailVerified,
	}

	_, err = a.repo.CreateUser(ctx, newUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user in new project")
		return
	}

	// Write audit event
	_ = a.writeAudit(ctx, createdTenant.ID, "", "USER", claims.UserID, "project.create", "tenant", createdTenant.ID, requestID(r), sourceIP(r), nil)

	project := ProjectResponse{
		ID:            createdTenant.ID,
		Slug:          createdTenant.Slug,
		Name:          createdTenant.Name,
		PrimaryRegion: createdTenant.PrimaryRegion,
		Role:          "OWNER",
		CreatedAt:     createdTenant.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	writeJSON(w, http.StatusCreated, project)
}

// handleGetProjectByID returns a specific project if the user has access
func (a *App) handleGetProjectByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	projectID := r.PathValue("projectID")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project ID is required")
		return
	}

	// For now, users can only access their own tenant
	if projectID != claims.TenantID {
		writeError(w, http.StatusForbidden, "access denied for this project")
		return
	}

	// Get tenant details
	tenant, err := a.repo.GetTenantByID(r.Context(), projectID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get project")
		return
	}

	// Get user's role
	user, err := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user info")
		return
	}

	project := ProjectResponse{
		ID:            tenant.ID,
		Slug:          tenant.Slug,
		Name:          tenant.Name,
		PrimaryRegion: tenant.PrimaryRegion,
		Role:          user.Role,
		CreatedAt:     tenant.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	writeJSON(w, http.StatusOK, project)
}
