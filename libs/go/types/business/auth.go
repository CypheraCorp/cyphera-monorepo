package business

import "github.com/cyphera/cyphera-api/libs/go/db"

// SignInRegisterData contains all data needed for sign-in or registration
type SignInRegisterData struct {
	Account    *db.Account
	User       *db.User
	Workspaces []db.Workspace
	IsNewUser  bool
}
