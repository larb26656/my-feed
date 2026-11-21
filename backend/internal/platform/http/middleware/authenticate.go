package middleware

import (
	"crypto/subtle"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

// Authenticate accepts Authorization: Bearer <access JWT> or, when apiKey is
// non-empty, Authorization: Bearer <apiKey>. JWT is tried first.
func Authenticate(jwtSecret, apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			_ = c.Error(apperr.Unauthorized.New("missing or invalid Authorization bearer token"))
			c.Abort()
			return
		}
		if err := validateAccessJWT(token, jwtSecret); err == nil {
			c.Next()
			return
		}
		if apiKey != "" && subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) == 1 {
			c.Next()
			return
		}
		_ = c.Error(apperr.Unauthorized.New("missing or invalid Authorization bearer token"))
		c.Abort()
	}
}

func validateAccessJWT(tokenString, secret string) error {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	claims := &jwt.RegisteredClaims{}
	token, err := parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return fmt.Errorf("invalid token")
	}
	if claims.Subject == "" {
		return fmt.Errorf("missing subject")
	}
	return nil
}
