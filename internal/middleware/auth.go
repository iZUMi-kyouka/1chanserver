package middleware

import (
	"1chanserver/internal/utils/utils_auth"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing access token; please login again",
			})
			return
		}

		accessToken := authHeader[len("Bearer "):]

		parsedToken, err := jwt.ParseWithClaims(accessToken, &utils_auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Return the HMAC secret key used to sign the parsedToken
			return []byte(utils_auth.JWT_SECRET_KEY), nil
		})

		claims, ok := parsedToken.Claims.(*utils_auth.Claims)

		log.Printf("parsedToken: %s; err: %s; claims: %v; ok: %s", parsedToken, err, claims, ok)
		switch {
		case err == nil && ok && parsedToken.Valid:
			log.Printf("Access token is valid.")
			claims, _ := parsedToken.Claims.(*utils_auth.Claims)
			c.Set("UserID", claims.UserID)
			c.Next()
		case err.Error() == "parsedToken has invalid claims: parsedToken is expired" && ok:
			refreshToken := c.GetHeader("Refresh-Token")

			if refreshToken == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"message": "refresh token missing; please login again",
				})
				return
			}

			log.Printf("Retrieving refresh parsedToken of user with id: %v\n", claims.UserID)

			err = utils_auth.ValidateRefreshToken(db, claims.UserID, refreshToken)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"message": fmt.Sprintf("invalid refresh token: %v", err.Error()),
				})
				return
			}

			newAccessToken, err := utils_auth.GenerateAccessToken(claims.UserID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"message": fmt.Sprintf("failed to generate new access token: %s", err.Error()),
				})
				return
			}

			c.SetCookie("Authorization", fmt.Sprintf("Bearer %s", newAccessToken), 60*10, "/", "", false, false)
			c.Set("UserID", claims.UserID)
			c.Next()
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": fmt.Sprintf("access token invalid: %s", err.Error()),
			})
		}
	}
}
