package db

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
