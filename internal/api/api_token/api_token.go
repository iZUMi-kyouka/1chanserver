package api_token

import (
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_auth"
	"1chanserver/internal/utils/utils_db"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
)

func RefreshToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		// Check if refresh token is available
		refreshToken := c.GetHeader("Refresh-Token")
		if refreshToken == "" {
			c.Error(
				api_error.New(errors.New("authorization header missing"), http.StatusUnauthorized, ""))
			return
		}

		// Check validity of refresh token
		parsedToken, err := jwt.ParseWithClaims(refreshToken, &utils_auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				err := api_error.NewFromStr("refresh token invalid", http.StatusUnauthorized)
				c.Header("X-RefreshToken", "failed")
				c.Error(err)
				return nil, err
			}

			return []byte(utils_auth.JWT_SECRET_KEY), nil
		})

		claims, ok := parsedToken.Claims.(*utils_auth.Claims)
		switch {
		case err == nil && ok && parsedToken.Valid:
			log.Printf("refresh token is valid")
			newAccessToken, err := utils_auth.GenerateAccessToken(claims.UserID)
			if err != nil {
				c.Header("X-RefreshToken", "failed")
				c.Error(err)
				return
			}

			// Check whether refresh token has been invalidated before its expiry
			storedHash, err := utils_db.FetchOne[string](
				db, "SELECT token_hash FROM refresh_tokens WHERE user_id = $1", claims.UserID)

			if err != nil {
				c.Header("X-RefreshToken", "failed")
				c.Error(api_error.New(err, http.StatusUnauthorized, "an internal error has occurred"))
				return
			}

			ok := utils_auth.VerifyArgon2Hash(refreshToken, storedHash)
			if !ok {
				c.Header("X-RefreshToken", "failed")
				c.Error(api_error.New(err, http.StatusUnauthorized, "refresh token invalid"))
				return
			}

			c.Header("X-RefreshToken", "success")
			c.JSON(http.StatusCreated, gin.H{
				"access_token": newAccessToken,
			})
		}
	}
}
