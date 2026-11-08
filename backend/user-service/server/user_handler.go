package server

import (
	"context"
	"errors"
	"strings"
	"time"

	userpb "gen/user-service-gen"

	db "github.com/FTNDreamTeam/FileSify/backend/user-service/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServer struct {
	userpb.UnimplementedUserServiceServer
	dbCollection *mongo.Collection
}

func NewUserServer(collection *mongo.Collection) *UserServer {
	return &UserServer{dbCollection: collection}
}

func (s *UserServer) RegisterUser(ctx context.Context, req *userpb.RegisterUserRequest) (*userpb.RegisterUserResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	firstName := strings.TrimSpace(req.GetFirstName())
	lastName := strings.TrimSpace(req.GetLastName())
	password := req.GetPassword()

	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validateName("first name", firstName); err != nil {
		return nil, err
	}
	if err := validateName("last name", lastName); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	var existingUser db.UserDBModel
	err := s.dbCollection.FindOne(ctx, bson.M{"email": email}).Decode(&existingUser)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "a user with this email already exists")
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, status.Errorf(codes.Internal, "failed to check existing user: %v", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	newUser := db.UserDBModel{
		ID:             primitive.NewObjectID(),
		Email:          email,
		Password:       string(hashedPassword),
		FirstName:      firstName,
		LastName:       lastName,
		PasswordActive: primitive.NewDateTimeFromTime(time.Now()),
	}

	_, err = s.dbCollection.InsertOne(ctx, newUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save user to database: %v", err)
	}

	responseUser := &userpb.UserProfile{
		Id:        newUser.ID.Hex(),
		Email:     newUser.Email,
		FirstName: newUser.FirstName,
		LastName:  newUser.LastName,
		PasswordValid: &date.Date{
			Year:  int32(newUser.PasswordActive.Time().Year()),
			Month: int32(newUser.PasswordActive.Time().Month()),
			Day:   int32(newUser.PasswordActive.Time().Day()),
		},
	}

	returnUser := &userpb.RegisterUserResponse{
		User:    responseUser,
		Message: "User registration successful",
	}

	println(returnUser)

	return returnUser, nil
}
