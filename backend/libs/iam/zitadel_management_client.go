package iam

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"workflow-engine/backend/libs/auth"
)

type zitadelUser struct {
	UserID             string `json:"userId"`
	Username           string `json:"username"`
	PreferredLoginName string `json:"preferredLoginName"`
	State              string `json:"state"`
	Details            struct {
		CreationDate string `json:"creationDate"`
		ChangeDate   string `json:"changeDate"`
	} `json:"details"`
	Human *struct {
		Profile struct {
			GivenName   string `json:"givenName"`
			FamilyName  string `json:"familyName"`
			DisplayName string `json:"displayName"`
		} `json:"profile"`
		Email struct {
			Email      string `json:"email"`
			IsVerified bool   `json:"isVerified"`
		} `json:"email"`
	} `json:"human"`
	Machine *struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"machine"`
}

type zitadelAuthorization struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	Project struct {
		ID string `json:"id"`
	} `json:"project"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	Roles []struct {
		Key         string `json:"key"`
		DisplayName string `json:"displayName"`
		Group       string `json:"group"`
	} `json:"roles"`
}

type zitadelProjectRole struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Group       string `json:"group"`
}

type zitadelPersonalAccessToken struct {
	CreationDate   string `json:"creationDate"`
	ChangeDate     string `json:"changeDate"`
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	OrganizationID string `json:"organizationId"`
	ExpirationDate string `json:"expirationDate"`
}

func (c *ZITADELManagementClient) ListUsers(ctx context.Context) ([]ManagedUser, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return nil, err
	}
	var usersResponse struct {
		Result []zitadelUser `json:"result"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/ListUsers", map[string]any{
		"query": map[string]any{"limit": "200"},
	}, &usersResponse); err != nil {
		return nil, err
	}
	authorizations, err := c.listAuthorizations(ctx, state.ProjectID)
	if err != nil {
		return nil, err
	}
	rolesByUser := map[string][]string{}
	for _, authorization := range authorizations {
		if authorization.Project.ID != state.ProjectID || !authorizationIsActive(authorization.State) {
			continue
		}
		for _, role := range authorization.Roles {
			rolesByUser[authorization.User.ID] = append(rolesByUser[authorization.User.ID], role.Key)
		}
	}
	users := make([]ManagedUser, 0, len(usersResponse.Result))
	for _, user := range usersResponse.Result {
		managed := managedUserFromZitadel(user)
		if !managedUserIsVisible(managed) {
			continue
		}
		managed.Roles = normalizeRoleKeys(rolesByUser[managed.ID])
		users = append(users, managed)
	}
	sort.Slice(users, func(i, j int) bool {
		return strings.ToLower(users[i].PreferredLoginName) < strings.ToLower(users[j].PreferredLoginName)
	})
	return users, nil
}

func (c *ZITADELManagementClient) GetUser(ctx context.Context, userID string) (ManagedUser, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return ManagedUser{}, err
	}
	var userResponse struct {
		User zitadelUser `json:"user"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/GetUserByID", map[string]any{
		"userId": userID,
	}, &userResponse); err != nil {
		return ManagedUser{}, err
	}
	managed := managedUserFromZitadel(userResponse.User)
	authorization, ok, err := c.findUserAuthorization(ctx, state.ProjectID, userID)
	if err != nil {
		return ManagedUser{}, err
	}
	if ok && authorizationIsActive(authorization.State) {
		for _, role := range authorization.Roles {
			managed.Roles = append(managed.Roles, role.Key)
		}
		managed.Roles = normalizeRoleKeys(managed.Roles)
	}
	return managed, nil
}

func (c *ZITADELManagementClient) CreateUser(ctx context.Context, input ManagedUserCreate) (ManagedUser, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return ManagedUser{}, err
	}
	username := strings.TrimSpace(input.Username)
	if username == "" {
		username = strings.TrimSpace(input.Email)
	}
	human := map[string]any{
		"profile": map[string]any{
			"givenName":         strings.TrimSpace(input.GivenName),
			"familyName":        strings.TrimSpace(input.FamilyName),
			"displayName":       displayName(input.GivenName, input.FamilyName),
			"preferredLanguage": "en",
		},
		"email": map[string]any{
			"email":      strings.TrimSpace(input.Email),
			"isVerified": true,
		},
		"password": map[string]any{
			"password":       input.Password,
			"changeRequired": input.PasswordChangeRequired,
		},
	}
	var createResponse struct {
		ID string `json:"id"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/CreateUser", map[string]any{
		"organizationId": state.OrgID,
		"username":       username,
		"human":          human,
	}, &createResponse); err != nil {
		return ManagedUser{}, err
	}
	if len(input.RoleKeys) > 0 {
		if err := c.setUserRoles(ctx, state, createResponse.ID, input.RoleKeys); err != nil {
			return ManagedUser{}, err
		}
	}
	return c.GetUser(ctx, createResponse.ID)
}

func (c *ZITADELManagementClient) ListClients(ctx context.Context) ([]ManagedClient, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return nil, err
	}
	var usersResponse struct {
		Result []zitadelUser `json:"result"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/ListUsers", map[string]any{
		"query": map[string]any{"limit": "500"},
	}, &usersResponse); err != nil {
		return nil, err
	}
	authorizations, err := c.listAuthorizations(ctx, state.ProjectID)
	if err != nil {
		return nil, err
	}
	rolesByUser := map[string][]string{}
	for _, authorization := range authorizations {
		if authorization.Project.ID != state.ProjectID || !authorizationIsActive(authorization.State) {
			continue
		}
		for _, role := range authorization.Roles {
			rolesByUser[authorization.User.ID] = append(rolesByUser[authorization.User.ID], role.Key)
		}
	}
	tokens, err := c.listPersonalAccessTokens(ctx, state.OrgID)
	if err != nil {
		return nil, err
	}
	tokensByUser := map[string][]ManagedClientTokenSummary{}
	for _, token := range tokens {
		tokensByUser[token.UserID] = append(tokensByUser[token.UserID], managedClientTokenSummaryFromZitadel(token))
	}
	clients := make([]ManagedClient, 0)
	for _, user := range usersResponse.Result {
		managed := managedClientFromZitadel(user)
		if !managedClientIsVisible(managed) {
			continue
		}
		roles := normalizeRoleKeys(rolesByUser[managed.ClientID])
		if !roleKeysContain(roles, auth.RoleWorkflowsaClient) {
			continue
		}
		managed.Role = auth.RoleWorkflowsaClient
		managed.Tokens = tokensByUser[managed.ClientID]
		sort.Slice(managed.Tokens, func(i, j int) bool {
			return managed.Tokens[i].TokenCreatedAt > managed.Tokens[j].TokenCreatedAt
		})
		clients = append(clients, managed)
	}
	sort.Slice(clients, func(i, j int) bool {
		return strings.ToLower(clients[i].Name) < strings.ToLower(clients[j].Name)
	})
	return clients, nil
}

func (c *ZITADELManagementClient) CreateClientToken(ctx context.Context, input ManagedClientTokenCreate) (ManagedClientToken, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return ManagedClientToken{}, err
	}
	username := strings.TrimSpace(input.Username)
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	clientMetadata := ManagedClient{
		Description: description,
		Environment: strings.TrimSpace(input.Environment),
		OwnerEmail:  strings.TrimSpace(input.OwnerEmail),
		Purpose:     strings.TrimSpace(input.Purpose),
	}
	machine := map[string]any{
		"name":            name,
		"description":     encodeClientDescription(clientMetadata),
		"accessTokenType": "ACCESS_TOKEN_TYPE_JWT",
	}
	payload := map[string]any{
		"organizationId": state.OrgID,
		"machine":        machine,
	}
	if username != "" {
		payload["username"] = username
	}
	var createResponse struct {
		ID string `json:"id"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/CreateUser", payload, &createResponse); err != nil {
		return ManagedClientToken{}, err
	}
	if err := c.setUserRoles(ctx, state, createResponse.ID, []string{auth.RoleWorkflowsaClient}); err != nil {
		_ = c.DeleteUser(ctx, createResponse.ID)
		return ManagedClientToken{}, err
	}
	tokenExpiresAt := clientTokenExpiration(input.TokenExpiresAt)
	var tokenResponse struct {
		CreationDate string `json:"creationDate"`
		TokenID      string `json:"tokenId"`
		Token        string `json:"token"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/AddPersonalAccessToken", map[string]any{
		"userId":         createResponse.ID,
		"expirationDate": tokenExpiresAt,
	}, &tokenResponse); err != nil {
		_ = c.DeleteUser(ctx, createResponse.ID)
		return ManagedClientToken{}, err
	}
	return ManagedClientToken{
		ClientID:       createResponse.ID,
		Username:       username,
		Name:           name,
		Description:    description,
		Environment:    strings.TrimSpace(input.Environment),
		OwnerEmail:     strings.TrimSpace(input.OwnerEmail),
		Purpose:        strings.TrimSpace(input.Purpose),
		Role:           auth.RoleWorkflowsaClient,
		TokenID:        tokenResponse.TokenID,
		Token:          tokenResponse.Token,
		TokenCreatedAt: tokenResponse.CreationDate,
		TokenExpiresAt: tokenExpiresAt,
	}, nil
}

func (c *ZITADELManagementClient) RotateClientToken(ctx context.Context, clientID string, input ManagedClientTokenRotate) (ManagedClientToken, error) {
	client, err := c.getClient(ctx, clientID)
	if err != nil {
		return ManagedClientToken{}, err
	}
	tokenExpiresAt := clientTokenExpiration(input.TokenExpiresAt)
	var tokenResponse struct {
		CreationDate string `json:"creationDate"`
		TokenID      string `json:"tokenId"`
		Token        string `json:"token"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/AddPersonalAccessToken", map[string]any{
		"userId":         client.ClientID,
		"expirationDate": tokenExpiresAt,
	}, &tokenResponse); err != nil {
		return ManagedClientToken{}, err
	}
	return ManagedClientToken{
		ClientID:       client.ClientID,
		Username:       client.Username,
		Name:           client.Name,
		Description:    client.Description,
		Environment:    client.Environment,
		OwnerEmail:     client.OwnerEmail,
		Purpose:        client.Purpose,
		Role:           auth.RoleWorkflowsaClient,
		TokenID:        tokenResponse.TokenID,
		Token:          tokenResponse.Token,
		TokenCreatedAt: tokenResponse.CreationDate,
		TokenExpiresAt: tokenExpiresAt,
	}, nil
}

func (c *ZITADELManagementClient) RevokeClientToken(ctx context.Context, clientID string, tokenID string) error {
	if _, err := c.getClient(ctx, clientID); err != nil {
		return err
	}
	return c.connect(ctx, "/zitadel.user.v2.UserService/RemovePersonalAccessToken", map[string]any{
		"userId":  strings.TrimSpace(clientID),
		"tokenId": strings.TrimSpace(tokenID),
	}, nil)
}

func (c *ZITADELManagementClient) DeleteClient(ctx context.Context, clientID string) error {
	if _, err := c.getClient(ctx, clientID); err != nil {
		return err
	}
	return c.DeleteUser(ctx, clientID)
}

func (c *ZITADELManagementClient) UpdateUser(ctx context.Context, userID string, input ManagedUserUpdate) (ManagedUser, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return ManagedUser{}, err
	}
	payload := map[string]any{"userId": userID}
	if strings.TrimSpace(input.Username) != "" {
		payload["username"] = strings.TrimSpace(input.Username)
	}
	human := map[string]any{}
	profile := map[string]any{}
	if strings.TrimSpace(input.GivenName) != "" {
		profile["givenName"] = strings.TrimSpace(input.GivenName)
	}
	if strings.TrimSpace(input.FamilyName) != "" {
		profile["familyName"] = strings.TrimSpace(input.FamilyName)
	}
	if strings.TrimSpace(input.DisplayName) != "" {
		profile["displayName"] = strings.TrimSpace(input.DisplayName)
	}
	if len(profile) > 0 {
		human["profile"] = profile
	}
	if strings.TrimSpace(input.Email) != "" {
		human["email"] = map[string]any{
			"email":      strings.TrimSpace(input.Email),
			"isVerified": true,
		}
	}
	if len(human) > 0 {
		payload["human"] = human
	}
	if len(payload) > 1 {
		if err := c.connect(ctx, "/zitadel.user.v2.UserService/UpdateUser", payload, nil); err != nil {
			return ManagedUser{}, err
		}
	}
	if input.UpdateRoleKeys {
		if err := c.setUserRoles(ctx, state, userID, input.RoleKeys); err != nil {
			return ManagedUser{}, err
		}
	}
	return c.GetUser(ctx, userID)
}

func (c *ZITADELManagementClient) TerminateUser(ctx context.Context, userID string) error {
	return c.connect(ctx, "/zitadel.user.v2.UserService/DeactivateUser", map[string]any{"userId": userID}, nil)
}

func (c *ZITADELManagementClient) ReactivateUser(ctx context.Context, userID string) error {
	return c.connect(ctx, "/zitadel.user.v2.UserService/ReactivateUser", map[string]any{"userId": userID}, nil)
}

func (c *ZITADELManagementClient) DeleteUser(ctx context.Context, userID string) error {
	return c.connect(ctx, "/zitadel.user.v2.UserService/DeleteUser", map[string]any{"userId": userID}, nil)
}

func (c *ZITADELManagementClient) ListRoles(ctx context.Context) ([]ManagedRole, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return nil, err
	}
	var response struct {
		ProjectRoles []zitadelProjectRole `json:"projectRoles"`
	}
	if err := c.connect(ctx, "/zitadel.project.v2.ProjectService/ListProjectRoles", map[string]any{
		"projectId":  state.ProjectID,
		"pagination": map[string]any{"limit": "200"},
	}, &response); err != nil {
		return nil, err
	}
	roles := make([]ManagedRole, 0, len(response.ProjectRoles))
	for _, role := range response.ProjectRoles {
		roles = append(roles, ManagedRole{Key: role.Key, DisplayName: role.DisplayName, Group: role.Group})
	}
	sort.Slice(roles, func(i, j int) bool {
		return strings.ToLower(roles[i].Key) < strings.ToLower(roles[j].Key)
	})
	return roles, nil
}

func (c *ZITADELManagementClient) CreateRole(ctx context.Context, input ManagedRoleCreate) (ManagedRole, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return ManagedRole{}, err
	}
	role := ManagedRole{Key: strings.TrimSpace(input.Key), DisplayName: strings.TrimSpace(input.DisplayName), Group: strings.TrimSpace(input.Group)}
	if role.Group == "" {
		role.Group = "Workflowsa"
	}
	if err := c.connect(ctx, "/zitadel.project.v2.ProjectService/AddProjectRole", map[string]any{
		"projectId":   state.ProjectID,
		"roleKey":     role.Key,
		"displayName": role.DisplayName,
		"group":       role.Group,
	}, nil); err != nil {
		return ManagedRole{}, err
	}
	return role, nil
}

func (c *ZITADELManagementClient) UpdateRole(ctx context.Context, roleKey string, input ManagedRoleUpdate) (ManagedRole, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return ManagedRole{}, err
	}
	payload := map[string]any{
		"projectId": state.ProjectID,
		"roleKey":   strings.TrimSpace(roleKey),
	}
	if strings.TrimSpace(input.DisplayName) != "" {
		payload["displayName"] = strings.TrimSpace(input.DisplayName)
	}
	payload["group"] = strings.TrimSpace(input.Group)
	if err := c.connect(ctx, "/zitadel.project.v2.ProjectService/UpdateProjectRole", payload, nil); err != nil {
		return ManagedRole{}, err
	}
	return ManagedRole{Key: strings.TrimSpace(roleKey), DisplayName: strings.TrimSpace(input.DisplayName), Group: strings.TrimSpace(input.Group)}, nil
}

func (c *ZITADELManagementClient) DeleteRole(ctx context.Context, roleKey string) error {
	state, err := c.readBootstrapState()
	if err != nil {
		return err
	}
	return c.connect(ctx, "/zitadel.project.v2.ProjectService/RemoveProjectRole", map[string]any{
		"projectId": state.ProjectID,
		"roleKey":   strings.TrimSpace(roleKey),
	}, nil)
}

func (c *ZITADELManagementClient) listAuthorizations(ctx context.Context, projectID string) ([]zitadelAuthorization, error) {
	var response struct {
		Authorizations []zitadelAuthorization `json:"authorizations"`
	}
	if err := c.connect(ctx, "/zitadel.authorization.v2.AuthorizationService/ListAuthorizations", map[string]any{
		"pagination": map[string]any{"limit": "500"},
	}, &response); err != nil {
		return nil, err
	}
	return response.Authorizations, nil
}

func (c *ZITADELManagementClient) listPersonalAccessTokens(ctx context.Context, organizationID string) ([]zitadelPersonalAccessToken, error) {
	var response struct {
		Result []zitadelPersonalAccessToken `json:"result"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/ListPersonalAccessTokens", map[string]any{
		"pagination":    map[string]any{"limit": "1000"},
		"sortingColumn": "PERSONAL_ACCESS_TOKEN_FIELD_NAME_CREATED_DATE",
	}, &response); err != nil {
		return nil, err
	}
	if strings.TrimSpace(organizationID) == "" {
		return response.Result, nil
	}
	tokens := make([]zitadelPersonalAccessToken, 0, len(response.Result))
	for _, token := range response.Result {
		if token.OrganizationID == "" || token.OrganizationID == organizationID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (c *ZITADELManagementClient) getClient(ctx context.Context, clientID string) (ManagedClient, error) {
	state, err := c.readBootstrapState()
	if err != nil {
		return ManagedClient{}, err
	}
	var userResponse struct {
		User zitadelUser `json:"user"`
	}
	if err := c.connect(ctx, "/zitadel.user.v2.UserService/GetUserByID", map[string]any{
		"userId": strings.TrimSpace(clientID),
	}, &userResponse); err != nil {
		return ManagedClient{}, err
	}
	client := managedClientFromZitadel(userResponse.User)
	if !managedClientIsVisible(client) {
		return ManagedClient{}, ErrZITADELManagedClientNotFound
	}
	authorization, ok, err := c.findUserAuthorization(ctx, state.ProjectID, client.ClientID)
	if err != nil {
		return ManagedClient{}, err
	}
	if !ok || !authorizationIsActive(authorization.State) {
		return ManagedClient{}, ErrZITADELManagedClientNotFound
	}
	roles := make([]string, 0, len(authorization.Roles))
	for _, role := range authorization.Roles {
		roles = append(roles, role.Key)
	}
	if !roleKeysContain(normalizeRoleKeys(roles), auth.RoleWorkflowsaClient) {
		return ManagedClient{}, ErrZITADELManagedClientNotFound
	}
	client.Role = auth.RoleWorkflowsaClient
	return client, nil
}

func (c *ZITADELManagementClient) findUserAuthorization(ctx context.Context, projectID string, userID string) (zitadelAuthorization, bool, error) {
	authorizations, err := c.listAuthorizations(ctx, projectID)
	if err != nil {
		return zitadelAuthorization{}, false, err
	}
	for _, authorization := range authorizations {
		if authorization.Project.ID == projectID && authorization.User.ID == userID {
			return authorization, true, nil
		}
	}
	return zitadelAuthorization{}, false, nil
}

func (c *ZITADELManagementClient) setUserRoles(ctx context.Context, state ZITADELBootstrapState, userID string, roleKeys []string) error {
	roleKeys = normalizeRoleKeys(roleKeys)
	authorization, exists, err := c.findUserAuthorization(ctx, state.ProjectID, userID)
	if err != nil {
		return err
	}
	if len(roleKeys) == 0 {
		if exists {
			return c.connect(ctx, "/zitadel.authorization.v2.AuthorizationService/DeleteAuthorization", map[string]any{"id": authorization.ID}, nil)
		}
		return nil
	}
	if exists {
		if err := c.connect(ctx, "/zitadel.authorization.v2.AuthorizationService/UpdateAuthorization", map[string]any{
			"id":       authorization.ID,
			"roleKeys": roleKeys,
		}, nil); err != nil {
			return err
		}
		if !authorizationIsActive(authorization.State) {
			return c.connect(ctx, "/zitadel.authorization.v2.AuthorizationService/ActivateAuthorization", map[string]any{"id": authorization.ID}, nil)
		}
		return nil
	}
	return c.connect(ctx, "/zitadel.authorization.v2.AuthorizationService/CreateAuthorization", map[string]any{
		"userId":         userID,
		"projectId":      state.ProjectID,
		"organizationId": state.OrgID,
		"roleKeys":       roleKeys,
	}, nil)
}

func managedUserFromZitadel(user zitadelUser) ManagedUser {
	managed := ManagedUser{
		ID:                 user.UserID,
		Username:           user.Username,
		PreferredLoginName: user.PreferredLoginName,
		State:              user.State,
		CreatedAt:          user.Details.CreationDate,
		ChangedAt:          user.Details.ChangeDate,
	}
	if managed.PreferredLoginName == "" {
		managed.PreferredLoginName = managed.Username
	}
	if user.Human != nil {
		managed.Type = "human"
		managed.GivenName = user.Human.Profile.GivenName
		managed.FamilyName = user.Human.Profile.FamilyName
		managed.DisplayName = user.Human.Profile.DisplayName
		managed.Email = user.Human.Email.Email
		managed.EmailVerified = user.Human.Email.IsVerified
	}
	if user.Machine != nil {
		managed.Type = "machine"
		managed.DisplayName = user.Machine.Name
	}
	if managed.DisplayName == "" {
		managed.DisplayName = managed.PreferredLoginName
	}
	return managed
}

func managedClientFromZitadel(user zitadelUser) ManagedClient {
	client := ManagedClient{
		ClientID:  user.UserID,
		Username:  user.Username,
		State:     user.State,
		CreatedAt: user.Details.CreationDate,
		ChangedAt: user.Details.ChangeDate,
	}
	if user.Machine != nil {
		client.Name = user.Machine.Name
		client.Description, client.Environment, client.OwnerEmail, client.Purpose = decodeClientDescription(user.Machine.Description)
	}
	if client.Name == "" {
		client.Name = user.PreferredLoginName
	}
	if client.Name == "" {
		client.Name = user.Username
	}
	return client
}

func managedUserIsVisible(user ManagedUser) bool {
	if user.Type == "machine" {
		return false
	}
	hiddenIdentities := []string{"workflow-login-client", "login-client", "workflowsa-bootstrap"}
	userIdentities := []string{user.Username, user.PreferredLoginName, user.DisplayName, user.Email}
	for _, userIdentity := range userIdentities {
		for _, hiddenIdentity := range hiddenIdentities {
			if strings.EqualFold(strings.TrimSpace(userIdentity), hiddenIdentity) {
				return false
			}
		}
	}
	return true
}

func managedClientIsVisible(client ManagedClient) bool {
	if strings.TrimSpace(client.ClientID) == "" {
		return false
	}
	hiddenIdentities := []string{"workflow-login-client", "login-client", "workflowsa-bootstrap"}
	clientIdentities := []string{client.Username, client.Name}
	for _, clientIdentity := range clientIdentities {
		for _, hiddenIdentity := range hiddenIdentities {
			if strings.EqualFold(strings.TrimSpace(clientIdentity), hiddenIdentity) {
				return false
			}
		}
	}
	return true
}

func managedClientTokenSummaryFromZitadel(token zitadelPersonalAccessToken) ManagedClientTokenSummary {
	status := "active"
	if expiresAt, err := time.Parse(time.RFC3339, token.ExpirationDate); err == nil && time.Now().UTC().After(expiresAt) {
		status = "expired"
	}
	return ManagedClientTokenSummary{
		TokenID:        token.ID,
		TokenCreatedAt: token.CreationDate,
		TokenChangedAt: token.ChangeDate,
		TokenExpiresAt: token.ExpirationDate,
		Status:         status,
	}
}

func encodeClientDescription(client ManagedClient) string {
	payload := map[string]string{
		"description": strings.TrimSpace(client.Description),
		"environment": strings.TrimSpace(client.Environment),
		"owner_email": strings.TrimSpace(client.OwnerEmail),
		"purpose":     strings.TrimSpace(client.Purpose),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return strings.TrimSpace(client.Description)
	}
	return "workflowsa-client:" + string(encoded)
}

func decodeClientDescription(value string) (string, string, string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", "", "", ""
	}
	raw := strings.TrimPrefix(trimmed, "workflowsa-client:")
	if !strings.HasPrefix(raw, "{") {
		return trimmed, "", "", ""
	}
	var payload struct {
		Description string `json:"description"`
		Environment string `json:"environment"`
		OwnerEmail  string `json:"owner_email"`
		Purpose     string `json:"purpose"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return trimmed, "", "", ""
	}
	return payload.Description, payload.Environment, payload.OwnerEmail, payload.Purpose
}

func roleKeysContain(values []string, role string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(role)) {
			return true
		}
	}
	return false
}

func normalizeRoleKeys(values []string) []string {
	seen := map[string]bool{}
	roles := make([]string, 0, len(values))
	for _, value := range values {
		role := strings.TrimSpace(value)
		if role == "" {
			continue
		}
		key := strings.ToLower(role)
		if seen[key] {
			continue
		}
		seen[key] = true
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func displayName(givenName string, familyName string) string {
	return strings.TrimSpace(strings.TrimSpace(givenName) + " " + strings.TrimSpace(familyName))
}

func clientTokenExpiration(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return time.Now().UTC().Add(365 * 24 * time.Hour).Format(time.RFC3339)
}

func authorizationIsActive(state string) bool {
	return state == "" || strings.EqualFold(state, "STATE_ACTIVE")
}
