package middleware

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_auth"
	"1chanserver/internal/utils/utils_handler"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || authHeader == "Bearer" {
			c.Error(
				api_error.NewFromStr("authorization header missing", http.StatusUnauthorized))
			c.Abort()
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

		claims, _ := parsedToken.Claims.(*utils_auth.Claims)

		//log.Printf("parsedToken: %s; err: %s; claims: %v;", parsedToken, err, claims)
		switch {
		case err == nil && parsedToken.Valid:
			//log.Printf("Access token is valid.")
			c.Set("UserID", claims.UserID)
			c.Next()
		default:
			c.Header("X-RefreshToken", "true")
			c.Error(api_error.NewFromStr("please relogin", http.StatusUnauthorized))
			c.Abort()
			return
		}
	}
}

func AuthResourceOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, userID := utils_handler.GetReqCx(c)

		var resource interface{}

		err := c.ShouldBindJSON(&resource)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		protectedResource, ok := resource.(models.Protected)
		if !ok {
			c.Error(api_error.NewFromErr(errors.New("invalid object"), http.StatusBadRequest))
			c.Abort()
			return
		}

		if !protectedResource.IsOwnedBy(&userID) {
			c.Error(api_error.NewFromStr("you cannot modify this resource", http.StatusForbidden))
			c.Abort()
			return
		}

		c.Next()
	}
}
