package models

type TokenClaims struct {
	Sub      string
	Email    string
	Username string
	Active   bool
}
