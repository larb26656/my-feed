package http

import (
	"time"

	"github.com/larb26656/my-feed/backend/internal/modules/auth/service"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func toTokenResponse(pair *service.TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    int64(pair.ExpiresIn / time.Second),
		TokenType:    "Bearer",
	}
}

type logoutResponse struct {
	OK bool `json:"ok"`
}
