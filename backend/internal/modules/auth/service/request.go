package service

// LoginInput is the Reader credential submitted at login.
type LoginInput struct {
	Username string
	Password string
}

// RefreshInput carries the opaque refresh token for rotation or logout.
type RefreshInput struct {
	RefreshToken string
}
