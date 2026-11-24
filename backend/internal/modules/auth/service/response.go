package service

import "time"

// TokenPair is the access and refresh credentials returned after login/refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}
