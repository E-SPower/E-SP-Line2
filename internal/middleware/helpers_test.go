package middleware

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// signForTest mints a token with the package's current secret. It goes through
// jwt directly rather than the auth service so the test stays in this package;
// the claims mirror service.Claims, which is what ParseToken decodes.
func signForTest(userID string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: "tester",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}
