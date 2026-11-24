package db

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role int

const (
	USER Role = iota
	ADMIN
	UKNOWN
)

func (r Role) String() string {
	switch r {
	case USER:
		return "user"
	case ADMIN:
		return "admin"
	default:
		return "uknown"
	}
}

func RoleFromString(s string) (Role, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case USER.String():
		return USER, nil
	case ADMIN.String():
		return ADMIN, nil
	default:
		return UKNOWN, fmt.Errorf("Error converting string to role: %q", s)
	}
}

type UserDBModel struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	Email          string             `bson:"email"`
	Password       string             `bson:"password"`
	FirstName      string             `bson:"first_name"`
	LastName       string             `bson:"last_name"`
	PasswordActive primitive.DateTime `bson:"password_active"` //When the password was created
	Verified       bool               `bson:"verified"`
	TwoFactor      bool               `bson:"two_factor"`
	StorageQuota   int32              `bson:"storage_quota"`
	UsedQuota      int32              `bson:"used_quota"`
	Role           Role               `bson:"role"`
}

type VerificationTokenModel struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	UserId            string             `bson:"user_id"`
	VerificationToken string             `bson:"verification_token"`
	Email             string             `bson:"email"`
	CreatedAt         time.Time          `bson:"created_at"`
}

type MagicLinkTokenModel struct {
	ID             primitive.ObjectID `bson:"_id"`
	UserId         string             `bson:"user_id"`
	MagicLinkToken string             `bson:"magic_link_token"`
	Email          string             `bson:"email"`
	CreatedAt      time.Time          `bson:"created_at"`
}
