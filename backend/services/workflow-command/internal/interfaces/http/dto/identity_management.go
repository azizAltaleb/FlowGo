package dto

type IdentityManagementUserResponse struct {
	ID                 string   `json:"id"`
	Username           string   `json:"username"`
	PreferredLoginName string   `json:"preferred_login_name"`
	DisplayName        string   `json:"display_name"`
	GivenName          string   `json:"given_name"`
	FamilyName         string   `json:"family_name"`
	Email              string   `json:"email"`
	EmailVerified      bool     `json:"email_verified"`
	State              string   `json:"state"`
	Type               string   `json:"type"`
	CreatedAt          string   `json:"created_at"`
	ChangedAt          string   `json:"changed_at"`
	Roles              []string `json:"roles"`
}

type IdentityManagementRoleResponse struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Group       string `json:"group"`
}

type IdentityManagementClientTokenResponse struct {
	ClientID       string `json:"client_id"`
	Username       string `json:"username"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Environment    string `json:"environment"`
	OwnerEmail     string `json:"owner_email"`
	Purpose        string `json:"purpose"`
	Role           string `json:"role"`
	TokenID        string `json:"token_id"`
	Token          string `json:"token"`
	TokenCreatedAt string `json:"token_created_at"`
	TokenExpiresAt string `json:"token_expires_at"`
}

type IdentityManagementClientTokenSummaryResponse struct {
	TokenID        string `json:"token_id"`
	TokenCreatedAt string `json:"token_created_at"`
	TokenChangedAt string `json:"token_changed_at"`
	TokenExpiresAt string `json:"token_expires_at"`
	Status         string `json:"status"`
}

type IdentityManagementClientResponse struct {
	ClientID    string                                         `json:"client_id"`
	Username    string                                         `json:"username"`
	Name        string                                         `json:"name"`
	Description string                                         `json:"description"`
	Environment string                                         `json:"environment"`
	OwnerEmail  string                                         `json:"owner_email"`
	Purpose     string                                         `json:"purpose"`
	Role        string                                         `json:"role"`
	State       string                                         `json:"state"`
	CreatedAt   string                                         `json:"created_at"`
	ChangedAt   string                                         `json:"changed_at"`
	Tokens      []IdentityManagementClientTokenSummaryResponse `json:"tokens"`
}

type ListIdentityManagementUsersResponse struct {
	Users []IdentityManagementUserResponse `json:"users"`
}

type ListIdentityManagementClientsResponse struct {
	Clients []IdentityManagementClientResponse `json:"clients"`
}

type ListIdentityManagementRolesResponse struct {
	Roles []IdentityManagementRoleResponse `json:"roles"`
}

type CreateIdentityManagementUserRequest struct {
	Username               string   `json:"username"`
	GivenName              string   `json:"given_name"`
	FamilyName             string   `json:"family_name"`
	Email                  string   `json:"email"`
	Password               string   `json:"password"`
	PasswordChangeRequired bool     `json:"password_change_required"`
	Roles                  []string `json:"roles"`
}

type CreateIdentityManagementClientTokenRequest struct {
	Username       string `json:"username"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Environment    string `json:"environment"`
	OwnerEmail     string `json:"owner_email"`
	Purpose        string `json:"purpose"`
	TokenExpiresAt string `json:"token_expires_at"`
}

type RotateIdentityManagementClientTokenRequest struct {
	TokenExpiresAt string `json:"token_expires_at"`
}

type UpdateIdentityManagementUserRequest struct {
	Username    string    `json:"username"`
	GivenName   string    `json:"given_name"`
	FamilyName  string    `json:"family_name"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	Roles       *[]string `json:"roles"`
}

type CreateIdentityManagementRoleRequest struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Group       string `json:"group"`
}

type UpdateIdentityManagementRoleRequest struct {
	DisplayName string `json:"display_name"`
	Group       string `json:"group"`
}
