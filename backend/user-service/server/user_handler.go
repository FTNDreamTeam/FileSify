package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
	"uuid"

	mailpb "gen/mail-service"
	userpb "gen/user-service-gen"

	db "github.com/FTNDreamTeam/FileSify/backend/user-service/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServer struct {
	userpb.UnimplementedUserServiceServer
	dbCollection           *mongo.Collection
	verificationCollection *mongo.Collection
	magicLinkCollection    *mongo.Collection
	mailClient             mailpb.MailServiceClient
}

func NewUserServer(
	collection *mongo.Collection,
	verificationCollection *mongo.Collection,
	magicLinkCollection *mongo.Collection,
	mailClient mailpb.MailServiceClient,
) *UserServer {
	return &UserServer{
		dbCollection:           collection,
		verificationCollection: verificationCollection,
		magicLinkCollection:    magicLinkCollection,
		mailClient:             mailClient,
	}
}

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

func getDummyHash() []byte {
	dummyHashOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-for-timing"), bcrypt.DefaultCost)
	})
	return dummyHash
}

func passwordMaxAge() time.Duration {
	d, err := time.ParseDuration(os.Getenv("PASSWORD_MAX_AGE"))
	if err != nil || d <= 0 {
		return 60 * 24 * time.Hour
	}
	return d
}

func EnsureTTLIndex(ctx context.Context, collection *mongo.Collection) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "created_at", Value: 1}},        //Key used for index
		Options: options.Index().SetExpireAfterSeconds(43200), //Expires after 12 hours
	}

	name, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create TTL index: %w", err)
	}

	log.Printf("Successfully created or verified TTL index: %s", name)
	return nil
}

func (s *UserServer) VerifyAccount(ctx context.Context, req *userpb.VerifyAccountRequest) (*userpb.VerifyAccountResponse, error) {
	token := strings.TrimPrefix(req.GetToken(), "veri")
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "verification token is required")
	}

	var verificationToken db.VerificationTokenModel
	err := s.verificationCollection.FindOne(ctx, bson.M{"verification_token": token}).Decode(&verificationToken)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			slog.Warn("verification_failure", "event", "auth.verify.failed", "reason", "invalid_token", "status", "failure")
			return nil, status.Error(codes.NotFound, "verification token is invalid or has expired")
		}
		slog.Error("verification_failure", "event", "auth.verify.failed", "reason", "db_error", "status", "failure", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "verification lookup failed: %v", err)
	}

	objectID, err := primitive.ObjectIDFromHex(verificationToken.UserId)
	if err != nil {
		slog.Error("verification_failure", "event", "auth.verify.failed", "reason", "invalid_user_id", "status", "failure", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "invalid user id in token: %v", err)
	}

	result, err := s.dbCollection.UpdateByID(ctx, objectID, bson.M{"$set": bson.M{"verified": true}})
	if err != nil {
		slog.Error("verification_failure", "event", "auth.verify.failed", "reason", "db_error", "status", "failure", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "failed to mark user as verified: %v", err)
	}
	if result.MatchedCount == 0 {
		slog.Error("verification_failure", "event", "auth.verify.failed", "reason", "user_not_found", "status", "failure")
		return nil, status.Error(codes.Internal, "user associated with token not found")
	}

	if _, err := s.verificationCollection.DeleteOne(ctx, bson.M{"_id": verificationToken.ID}); err != nil {
		slog.Warn("verification_cleanup_failed", "event", "auth.verify.cleanup", "error", err.Error())
	}

	slog.Info("verification_success", "event", "auth.verify", "user_id", verificationToken.UserId, "status", "success")

	return &userpb.VerifyAccountResponse{
		Message: "User Verification Successful",
	}, nil
}

func (s *UserServer) RequestMagicLink(ctx context.Context, req *userpb.RequestMagicLinkRequest) (*userpb.RequestMagicLinkResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	const genericMessage = "If an account exists for that email, a sign-in link has been sent."

	var user db.UserDBModel
	err := s.dbCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			slog.Info("magic_link_requested", "event", "auth.magic_link.request", "status", "no_user")
			return &userpb.RequestMagicLinkResponse{Message: genericMessage}, nil
		}
		slog.Error("magic_link_request_failed", "event", "auth.magic_link.request", "status", "db_error", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "magic link lookup failed: %v", err)
	}

	if !user.Verified {
		slog.Info("magic_link_requested", "event", "auth.magic_link.request", "status", "unverified", "user_id", user.ID.Hex())
		return &userpb.RequestMagicLinkResponse{Message: genericMessage}, nil
	}

	token := uuid.New().String()
	tokenModel := &db.MagicLinkTokenModel{
		ID:             primitive.NewObjectID(),
		UserId:         user.ID.Hex(),
		MagicLinkToken: token,
		Email:          user.Email,
		CreatedAt:      time.Now(),
	}

	if _, err := s.magicLinkCollection.InsertOne(ctx, tokenModel); err != nil {
		slog.Error("magic_link_request_failed", "event", "auth.magic_link.request", "status", "db_error", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "failed to save magic link token: %v", err)
	}

	magicLinkURL := "http://localhost:8080/HTML/magic-login.html?token=" + token

	go func() {
		mailCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, mailErr := s.mailClient.SendMagicLink(mailCtx, &userpb.SendMagicLinkRequest{
			Email:         user.Email,
			Username:      user.FirstName,
			MagicLinkUrl:  magicLinkURL,
			ExpiresAtUnix: time.Now().Add(12 * time.Hour).Unix(),
		})
		if mailErr != nil {
			slog.Error("magic_link_email_failed", "event", "auth.magic_link.request", "status", "mail_failed", "user_id", user.ID.Hex(), "error", mailErr.Error())
		}
	}()

	slog.Info("magic_link_requested", "event", "auth.magic_link.request", "status", "success", "user_id", user.ID.Hex())

	return &userpb.RequestMagicLinkResponse{Message: genericMessage}, nil
}

func (s *UserServer) VerifyMagicLink(ctx context.Context, req *userpb.VerifyMagicLinkRequest) (*userpb.LoginUserResponse, error) {
	token := strings.TrimSpace(req.GetToken())
	if token == "" {
		slog.Warn("magic_link_failure", "event", "auth.magic_link.failed", "reason", "invalid_input", "status", "failure")
		return nil, status.Error(codes.InvalidArgument, "magic link token is required")
	}

	var tokenDoc db.MagicLinkTokenModel
	err := s.magicLinkCollection.FindOne(ctx, bson.M{"magic_link_token": token}).Decode(&tokenDoc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			slog.Warn("magic_link_failure", "event", "auth.magic_link.failed", "reason", "invalid_token", "status", "failure")
			return nil, status.Error(codes.NotFound, "magic link is invalid or has expired")
		}
		slog.Error("magic_link_failure", "event", "auth.magic_link.failed", "reason", "db_error", "status", "failure", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "magic link lookup failed: %v", err)
	}

	objectID, err := primitive.ObjectIDFromHex(tokenDoc.UserId)
	if err != nil {
		slog.Error("magic_link_failure", "event", "auth.magic_link.failed", "reason", "invalid_user_id", "status", "failure", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "invalid user id in token: %v", err)
	}

	var user db.UserDBModel
	err = s.dbCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		slog.Error("magic_link_failure", "event", "auth.magic_link.failed", "reason", "user_not_found", "status", "failure", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "user not found for token: %v", err)
	}

	if _, err := s.magicLinkCollection.DeleteOne(ctx, bson.M{"_id": tokenDoc.ID}); err != nil {
		slog.Warn("magic_link_cleanup_failed", "event", "auth.magic_link.cleanup", "error", err.Error())
	}

	jwtToken, err := GenerateToken(user.ID.Hex(), user.Email)
	if err != nil {
		slog.Error("magic_link_failure", "event", "auth.magic_link.failed", "reason", "token_generation_failed", "status", "failure", "user_id", user.ID.Hex(), "error", err.Error())
		return nil, status.Errorf(codes.Internal, "failed to issue token: %v", err)
	}

	slog.Info("magic_link_success", "event", "auth.magic_link", "user_id", user.ID.Hex(), "status", "success")

	return &userpb.LoginUserResponse{
		Token: jwtToken,
		User: &userpb.UserProfile{
			Id:        user.ID.Hex(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			PasswordValid: &date.Date{
				Year:  int32(user.PasswordActive.Time().Year()),
				Month: int32(user.PasswordActive.Time().Month()),
				Day:   int32(user.PasswordActive.Time().Day()),
			},
		},
	}, nil
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
		Verified:       false,
	}

	verificationToken := uuid.New().String()
	verificationLink := "http://localhost:8080/HTML/verify-email.html?token=veri" + verificationToken
	expiresAt := time.Now().Add(12 * time.Hour).Unix()

	tokenModel := &db.VerificationTokenModel{
		ID:                primitive.NewObjectID(),
		UserId:            newUser.ID.Hex(),
		VerificationToken: verificationToken,
		Email:             newUser.Email,
		CreatedAt:         time.Now(),
	}

	_, err = s.verificationCollection.InsertOne(ctx, tokenModel)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save token to verification database: %v", err)
	}

	go func() {
		mailCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, mailErr := s.mailClient.SendVerification(mailCtx, &userpb.SendVerificationRequest{
			Email:           newUser.Email,
			Username:        newUser.FirstName,
			VerificationUrl: verificationLink,
			ExpiresAtUnix:   expiresAt,
		})

		if mailErr != nil {
			log.Printf("Asynchronous email dispatch failed for %s: %v", newUser.Email, mailErr)
		}

	}()

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

func (s *UserServer) LoginUser(ctx context.Context, req *userpb.LoginUserRequest) (*userpb.LoginUserResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	password := req.GetPassword()

	if email == "" || password == "" {
		slog.Warn("login_failure", "event", "auth.login.failed", "reason", "invalid_input", "status", "failure")
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	var user db.UserDBModel
	err := s.dbCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			_ = bcrypt.CompareHashAndPassword(getDummyHash(), []byte(password))
			slog.Warn("login_failure", "event", "auth.login.failed", "reason", "no_user", "status", "failure")
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		log.Fatalf("Database error: %v", err)
		return nil, status.Errorf(codes.Internal, "login lookup failed: %v", err)

	}

	if !user.Verified {
		slog.Warn("login_failure", "event", "auth.login.failed", "reason", "account_not_verified", "user_id", user.ID.Hex(), "status", "failure")
		return nil, status.Error(codes.FailedPrecondition, "your account is not verified. please check your email for the verification link")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		log.Fatalf("Token generating error: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	token, err := GenerateToken(user.ID.Hex(), user.Email)
	if err != nil {
		slog.Error("login_failure", "event", "auth.login.failed", "reason", "token_generation_failed", "user_id", user.ID.Hex(), "status", "failure", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "failed to issue token: %v", err)
	}

	if time.Since(user.PasswordActive.Time()) > passwordMaxAge() {
		slog.Warn("login_failure", "event", "auth.login.failed", "reason", "password_expired", "user_id", user.ID.Hex(), "status", "failure")
		return nil, status.Error(codes.FailedPrecondition, "your password has expired; please reset it to continue")
	}

	slog.Info("login_success", "event", "auth.login", "user_id", user.ID.Hex(), "status", "success")

	return &userpb.LoginUserResponse{
		Token: token,
		User: &userpb.UserProfile{
			Id:        user.ID.Hex(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			PasswordValid: &date.Date{
				Year:  int32(user.PasswordActive.Time().Year()),
				Month: int32(user.PasswordActive.Time().Month()),
				Day:   int32(user.PasswordActive.Time().Day()),
			},
		},
	}, nil
}
