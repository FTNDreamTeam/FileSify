package server

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/FTNDreamTeam/FileSify/backend/user-service/db"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

const tokenTTL = 24 * time.Hour

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		panic("JWT_SECRET is not set")
	}
	return []byte(s)
}

func GenerateToken(userID, email string, role db.Role) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role.String(),
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(tokenTTL).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(jwtSecret())
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func ParseAndValidateToken(tokenString string) (string, db.Role, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret(), nil
	})

	if err != nil {
		log.Println("err at jwt.ParseWithClaims")
		return "", db.UKNOWN, fmt.Errorf("Invalid token: %v", err)
	}

	if !token.Valid {
		return "", db.UKNOWN, errors.New("Token is not valid")
	}

	userID := claims.UserID
	if userID == "" {
		userID = claims.Subject
	}
	if userID == "" {
		return "", db.UKNOWN, errors.New("UserId not found in token claims")
	}
	role, _ := db.RoleFromString(claims.Role)
	if !role.IsValid() {
		return userID, db.UKNOWN, errors.New("Invalid or missing role in token claims")
	}

	fmt.Println("userId" + userID)
	return userID, role, nil
}
