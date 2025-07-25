package helpers

import (
	"encoding/json"
	"log"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// AccountResponse represents the standardized API response for account operations
type AccountResponse struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	Name               string                 `json:"name"`
	AccountType        string                 `json:"account_type"`
	BusinessName       string                 `json:"business_name,omitempty"`
	BusinessType       string                 `json:"business_type,omitempty"`
	WebsiteURL         string                 `json:"website_url,omitempty"`
	SupportEmail       string                 `json:"support_email,omitempty"`
	SupportPhone       string                 `json:"support_phone,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	FinishedOnboarding bool                   `json:"finished_onboarding"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
	Workspaces         []WorkspaceResponse    `json:"workspaces,omitempty"`
}

// AccountDetailsResponse represents detailed account information with user and wallets
type AccountDetailsResponse struct {
	AccountResponse AccountResponse  `json:"account"`
	User            UserResponse     `json:"user,omitempty"`
	Wallets         []WalletResponse `json:"wallets,omitempty"`
}

// WorkspaceResponse represents workspace information in account responses
type WorkspaceResponse struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
	UpdatedAt   int64                  `json:"updated_at"`
}

// WalletResponse represents wallet information in account responses
type WalletResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	WalletType    string                 `json:"wallet_type"`
	WalletAddress string                 `json:"wallet_address"`
	NetworkType   string                 `json:"network_type"`
	NetworkID     string                 `json:"network_id,omitempty"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// ToAccountResponse converts database account model to API response
func ToAccountResponse(account db.Account, workspaces []db.Workspace) AccountResponse {
	workspacesResponses := make([]WorkspaceResponse, 0)
	for _, workspace := range workspaces {
		workspacesResponses = append(workspacesResponses, ToWorkspaceResponse(workspace))
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(account.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling account metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return AccountResponse{
		ID:                 account.ID.String(),
		Object:             "account",
		Name:               account.Name,
		AccountType:        string(account.AccountType),
		BusinessName:       account.BusinessName.String,
		BusinessType:       account.BusinessType.String,
		WebsiteURL:         account.WebsiteUrl.String,
		SupportEmail:       account.SupportEmail.String,
		SupportPhone:       account.SupportPhone.String,
		Metadata:           metadata,
		FinishedOnboarding: account.FinishedOnboarding.Bool,
		CreatedAt:          account.CreatedAt.Time.Unix(),
		UpdatedAt:          account.UpdatedAt.Time.Unix(),
		Workspaces:         workspacesResponses,
	}
}

// ToAccountDetailsResponse converts account access data to detailed response
func ToAccountDetailsResponse(account db.Account, user db.User, workspaces []db.Workspace) AccountDetailsResponse {
	return AccountDetailsResponse{
		AccountResponse: ToAccountResponse(account, workspaces),
		User:            ToUserResponse(user),
	}
}

// ToWorkspaceResponse converts database workspace model to API response
func ToWorkspaceResponse(workspace db.Workspace) WorkspaceResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(workspace.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling workspace metadata: %v", err)
		metadata = make(map[string]interface{})
	}

	return WorkspaceResponse{
		ID:          workspace.ID.String(),
		Object:      "workspace",
		Name:        workspace.Name,
		Description: workspace.Description.String,
		Metadata:    metadata,
		CreatedAt:   workspace.CreatedAt.Time.Unix(),
		UpdatedAt:   workspace.UpdatedAt.Time.Unix(),
	}
}

// ToWalletResponse converts database wallet model to API response
func ToWalletResponse(wallet db.Wallet) WalletResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(wallet.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling wallet metadata: %v", err)
		metadata = make(map[string]interface{})
	}

	var networkID string
	if wallet.NetworkID.Valid {
		networkUUID := uuid.UUID(wallet.NetworkID.Bytes)
		networkID = networkUUID.String()
	}

	return WalletResponse{
		ID:            wallet.ID.String(),
		Object:        "wallet",
		WalletType:    wallet.WalletType,
		WalletAddress: wallet.WalletAddress,
		NetworkType:   string(wallet.NetworkType),
		NetworkID:     networkID,
		Nickname:      wallet.Nickname.String,
		ENS:           wallet.Ens.String,
		IsPrimary:     wallet.IsPrimary.Bool,
		Verified:      wallet.Verified.Bool,
		Metadata:      metadata,
		CreatedAt:     wallet.CreatedAt.Time.Unix(),
		UpdatedAt:     wallet.UpdatedAt.Time.Unix(),
	}
}
