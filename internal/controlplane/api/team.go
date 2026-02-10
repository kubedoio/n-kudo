package controlplane

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// CreateInvitationRequest represents a request to invite a user
type CreateInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"` // ADMIN, OPERATOR, VIEWER
}

// InvitationResponse represents an invitation in the response
type InvitationResponse struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	InvitedByUserID string    `json:"invited_by_user_id"`
	Status          string    `json:"status"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// handleCreateInvitation creates a new project invitation
func (a *App) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Validate role
	validRoles := map[string]bool{"ADMIN": true, "OPERATOR": true, "VIEWER": true}
	if !validRoles[req.Role] {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	// Check if user already has access to this project
	existingUser, _ := a.repo.GetUserByEmail(r.Context(), req.Email)
	if existingUser.TenantID == claims.TenantID {
		writeError(w, http.StatusConflict, "user already has access to this project")
		return
	}

	// Generate invitation token
	token, err := generateRandomString(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	tokenHash := hashToken(token)

	// Create invitation (expires in 7 days)
	invitation := store.ProjectInvitation{
		ID:              uuid.New().String(),
		TenantID:        claims.TenantID,
		Email:           req.Email,
		Role:            req.Role,
		InvitedByUserID: claims.UserID,
		TokenHash:       tokenHash,
		Status:          "PENDING",
		ExpiresAt:       time.Now().Add(7 * 24 * time.Hour),
	}

	if err := a.repo.CreateInvitation(r.Context(), invitation); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}

	// Get inviter info for email
	inviter, _ := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	inviterName := inviter.DisplayName
	if inviterName == "" {
		inviterName = inviter.Email
	}

	// Get project info
	tenant, _ := a.repo.GetTenantByID(r.Context(), claims.TenantID)

	// Send invitation email
	if a.emailService != nil && a.emailService.IsEnabled() {
		if err := a.emailService.SendInvitationEmail(req.Email, inviterName, tenant.Name, token); err != nil {
			log.Printf("Failed to send invitation email: %v", err)
			// Don't fail the request, but log the error
		}
	}

	// Write audit event
	_ = a.writeAudit(r.Context(), claims.TenantID, "", "USER", claims.UserID, "invitation.create", "invitation", invitation.ID, requestID(r), sourceIP(r), nil)

	writeJSON(w, http.StatusCreated, InvitationResponse{
		ID:              invitation.ID,
		Email:           invitation.Email,
		Role:            invitation.Role,
		InvitedByUserID: invitation.InvitedByUserID,
		Status:          invitation.Status,
		ExpiresAt:       invitation.ExpiresAt,
		CreatedAt:       invitation.CreatedAt,
	})
}

// handleListInvitations lists pending invitations for the current project
func (a *App) handleListInvitations(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	invitations, err := a.repo.ListPendingInvitations(r.Context(), claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}

	var response []InvitationResponse
	for _, inv := range invitations {
		response = append(response, InvitationResponse{
			ID:              inv.ID,
			Email:           inv.Email,
			Role:            inv.Role,
			InvitedByUserID: inv.InvitedByUserID,
			Status:          inv.Status,
			ExpiresAt:       inv.ExpiresAt,
			CreatedAt:       inv.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"invitations": response,
	})
}

// handleCancelInvitation cancels a pending invitation
func (a *App) handleCancelInvitation(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	invitationID := r.PathValue("invitationID")
	if invitationID == "" {
		writeError(w, http.StatusBadRequest, "invitation ID is required")
		return
	}

	if err := a.repo.CancelInvitation(r.Context(), claims.TenantID, invitationID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel invitation")
		return
	}

	// Write audit event
	_ = a.writeAudit(r.Context(), claims.TenantID, "", "USER", claims.UserID, "invitation.cancel", "invitation", invitationID, requestID(r), sourceIP(r), nil)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Invitation cancelled",
	})
}

// handleGetMyInvitations lists invitations for the current user
func (a *App) handleGetMyInvitations(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Get user email
	user, err := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user info")
		return
	}

	invitations, err := a.repo.ListUserInvitations(r.Context(), user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"invitations": invitations,
	})
}

// handleAcceptInvitation accepts an invitation via token
func (a *App) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	tokenHash := hashToken(req.Token)

	// Get invitation
	invitation, err := a.repo.GetInvitationByToken(r.Context(), tokenHash)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "invalid or expired invitation")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get invitation")
		return
	}

	// Check if invitation is still valid
	if invitation.Status != "PENDING" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invitation is %s", invitation.Status))
		return
	}

	if invitation.ExpiresAt.Before(time.Now()) {
		writeError(w, http.StatusBadRequest, "invitation has expired")
		return
	}

	// Check if user exists with this email
	existingUser, err := a.repo.GetUserByEmail(r.Context(), invitation.Email)
	if err != nil {
		// User doesn't exist - they need to register first
		writeError(w, http.StatusBadRequest, "you need to create an account first. Please register with this email address.")
		return
	}

	// Create user in the new project (invitation project)
	newUser := store.User{
		ID:            uuid.New().String(),
		TenantID:      invitation.TenantID,
		Email:         existingUser.Email,
		DisplayName:   existingUser.DisplayName,
		Role:          invitation.Role,
		PasswordHash:  existingUser.PasswordHash, // Same password
		IsActive:      true,
		EmailVerified: existingUser.EmailVerified,
	}

	if _, err := a.repo.CreateUser(r.Context(), newUser); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add user to project")
		return
	}

	// Mark invitation as accepted
	if err := a.repo.AcceptInvitation(r.Context(), invitation.ID, newUser.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to accept invitation")
		return
	}

	// Write audit event
	_ = a.writeAudit(r.Context(), invitation.TenantID, "", "USER", newUser.ID, "invitation.accept", "invitation", invitation.ID, requestID(r), sourceIP(r), nil)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Invitation accepted successfully. You can now switch to this project.",
	})
}

// handleDeclineInvitation declines an invitation
func (a *App) handleDeclineInvitation(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	invitationID := r.PathValue("invitationID")
	if invitationID == "" {
		writeError(w, http.StatusBadRequest, "invitation ID is required")
		return
	}

	// Get user email
	user, err := a.repo.GetUserByID(r.Context(), claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user info")
		return
	}

	// Get invitation to verify it belongs to this user
	invitation, err := a.repo.GetInvitationByToken(r.Context(), invitationID)
	if err != nil {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}

	if invitation.Email != user.Email {
		writeError(w, http.StatusForbidden, "this invitation is not for you")
		return
	}

	if err := a.repo.DeclineInvitation(r.Context(), invitationID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decline invitation")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Invitation declined",
	})
}
