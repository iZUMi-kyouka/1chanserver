package utils_auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/argon2"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
)

type Claims struct {
	UserID uuid.UUID `json:"id"`
	jwt.RegisteredClaims
}

var JWT_SECRET_KEY = []byte(os.Getenv("JWT_SECRET_KEY"))

const (
	ARGON2_TIME       = uint32(1)
	ARGON2_MEMORY     = uint32(64 * 1024)
	ARGON2_THREADS    = uint8(2)
	ARGON2_KEYLENGTH  = uint32(32)
	ARGON2_SALTLENGTH = uint32(16)

	JWT_ACCESS_TOKEN_EXPIRATION  = 10 * time.Minute
	JWT_REFRESH_TOKEN_EXPIRATION = 14 * 24 * time.Hour
)

// formatHash formats the generated hash and salt into a conventional format
// for storage
func formatHash(salt []byte, hashedPassword []byte) string {
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHashedPassword := base64.RawStdEncoding.EncodeToString(hashedPassword)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		uint32(argon2.Version),
		ARGON2_MEMORY,
		ARGON2_TIME,
		ARGON2_THREADS,
		encodedSalt,
		encodedHashedPassword,
	)
}

func parsePasswordHash(passwordHash *string) (
	uint32, uint32, uint8, string, string, error) {
	pattern := fmt.Sprintf(
		"^\\$argon2id\\$v=%d\\$m=(\\d+),t=(\\d+),p=(\\d+)\\$([A-Za-z0-9+/=]+)\\$([A-Za-z0-9+/=]+)$",
		uint32(argon2.Version))
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(*passwordHash)

	if matches == nil {
		return 0, 0, 0, "", "", errors.New("Invalid argon2 hash format.")
	}

	arg2Mem, _ := strconv.ParseUint(matches[1], 10, 32)
	arg2Time, _ := strconv.ParseUint(matches[2], 10, 32)
	arg2Threads, _ := strconv.ParseUint(matches[3], 10, 32)

	return uint32(arg2Mem), uint32(arg2Time), uint8(arg2Threads), matches[4], matches[5], nil
}

func generateArgon2Salt() []byte {
	salt := make([]byte, ARGON2_SALTLENGTH)
	if _, err := rand.Read(salt); err != nil {
		log.Fatalf("error generating salt: %v", err)
	}

	return salt
}

func generateArgon2Hash(payload []byte, salt []byte) []byte {
	return argon2.IDKey(payload, salt, ARGON2_TIME, ARGON2_MEMORY, ARGON2_THREADS, ARGON2_KEYLENGTH)
}

func GenerateArgon2Hash(payload string) string {
	salt := generateArgon2Salt()
	hash := generateArgon2Hash([]byte(payload), salt)
	return formatHash(salt, hash)
}

func VerifyArgon2Hash(payload string, storedHash string) bool {
	arg2Mem, arg2Time, arg2Threads, salt, expectedHash, err := parsePasswordHash(&storedHash)
	if err != nil {
		return false
	}

	decodedSalt, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		log.Fatalf("error decoding salt: %v", err)
	}

	computedHash := base64.RawStdEncoding.EncodeToString(
		argon2.IDKey([]byte(payload), decodedSalt, arg2Time, arg2Mem, arg2Threads, ARGON2_KEYLENGTH))

	log.Printf("computedHash: %s | storedHash: %s", computedHash, expectedHash)
	return computedHash == expectedHash
}

func GenerateAccessToken(userID uuid.UUID) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWT_ACCESS_TOKEN_EXPIRATION)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWT_SECRET_KEY)
}

func GenerateRefreshToken(userID uuid.UUID) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWT_REFRESH_TOKEN_EXPIRATION)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWT_SECRET_KEY)
}

func HashRefreshToken(refreshToken string) string {
	salt := generateArgon2Salt()
	hash := generateArgon2Hash([]byte(refreshToken), salt)
	return formatHash(salt, hash)
}

func ValidateRefreshToken(db *sqlx.DB, userID uuid.UUID, givenRefreshToken string) error {
	var storedHash string
	err := db.Get(&storedHash, "SELECT token_hash FROM refresh_tokens WHERE user_id = $1", userID)

	if err != nil {
		return err
	}

	ok := VerifyArgon2Hash(givenRefreshToken, storedHash)
	if !ok {
		return errors.New("invalid refresh token.")
	}

	return nil
}

func SetAccessAndRefreshToken(c *gin.Context, refreshToken string, accessToken string) {
	c.SetCookie("Refresh-Token", refreshToken, 3600*24*14, "/", "", false, true)
	c.SetCookie("Authorization", fmt.Sprintf("Bearer %s", accessToken), 60*10, "/", "", false, false)
}