package server

import (
	"context"
	"errors"
	"time"

	db "github.com/FTNDreamTeam/FileSify/backend/user-service/db"
	userpb "github.com/FTNDreamTeam/FileSify/backend/user-service/proto/user-service-gen"
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
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	var existingUser db.UserDBModel
	err := s.dbCollection.FindOne(ctx, bson.M{"email": req.GetEmail()}).Decode(&existingUser)
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
		Email:          req.GetEmail(),
		Password:       string(hashedPassword),
		FirstName:      req.GetFirstName(),
		LastName:       req.GetLastName(),
		PasswordActive: primitive.NewDateTimeFromTime(time.Now()),
	}

	_, err = s.dbCollection.InsertOne(ctx, newUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save user to database: %v", err)
	}

	responseUser := &userpb.UserProfile{
		Id:        newUser.ID.Hex(), // Converts ObjectID to string
		Email:     newUser.Email,
		FirstName: newUser.FirstName,
		LastName:  newUser.LastName,
		PasswordValid: &date.Date{
			Year:  int32(newUser.PasswordActive.Time().Year()),
			Month: int32(newUser.PasswordActive.Time().Month()),
			Day:   int32(newUser.PasswordActive.Time().Day()),
		},
	}

	return &userpb.RegisterUserResponse{
		User:    responseUser,
		Message: "User registration successful",
	}, nil
}
