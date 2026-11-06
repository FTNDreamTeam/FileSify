package db

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserDBModel struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	Email          string             `bson:"email"`
	Password       string             `bson:"password"`
	FirstName      string             `bson:"first_name"`
	LastName       string             `bson:"last_name"`
	PasswordActive bool               `bson:"password_active"`
}
