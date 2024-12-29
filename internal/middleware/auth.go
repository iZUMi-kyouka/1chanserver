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
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(
				api_error.New(errors.New("authorization header missing"), http.StatusUnauthorized, ""))
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
			c.Set("UserID", claims.UserID)
			c.Next()
		default:
			c.Header("X-RefreshToken", "true")
			c.Error(api_error.NewFromStr("please relogin", http.StatusUnauthorized))
			return
			//case err.Error() == "parsedToken has invalid claims: parsedToken is expired" && ok:
			//	refreshToken := c.GetHeader("Refresh-Token")
			//
			//	if refreshToken == "" {
			//		c.Error(api_error.New(errors.New("refresh token missing"), http.StatusUnauthorized, ""))
			//		return
			//	}
			//
			//	log.Printf("Retrieving refresh token of user with id: %v\n", claims.UserID)
			//
			//	err = utils_auth.ValidateRefreshToken(db, claims.UserID, refreshToken)
			//	if err != nil {
			//		c.Error(api_error.New(errors.New("refresh token invalid"), http.StatusUnauthorized, ""))
			//		return
			//	}
			//
			//	newAccessToken, err := utils_auth.GenerateAccessToken(claims.UserID)
			//	if err != nil {
			//		c.Error(err)
			//		return
			//	}
			//
			//	c.SetCookie("Authorization", fmt.Sprintf("Bearer %s", newAccessToken), 60*10, "/", "", false, false)
			//	c.Set("UserID", claims.UserID)
			//	c.Next()
			//default:
			//	c.Error(api_error.New(errors.New("access token invalid"), http.StatusUnauthorized, ""))
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
			return
		}

		protectedResource, ok := resource.(models.Protected)
		if !ok {
			c.Error(api_error.NewC(errors.New("invalid object"), http.StatusBadRequest))
			return
		}

		if !protectedResource.IsOwnedBy(&userID) {
			c.Error(api_error.NewFromStr("you cannot modify this resource", http.StatusForbidden))
			return
		}

		c.Next()
	}
}
