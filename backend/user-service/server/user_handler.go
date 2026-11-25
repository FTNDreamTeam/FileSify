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

	storagepb "gen/data-storage-service-gen"
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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type UserServer struct {
	userpb.UnimplementedUserServiceServer
	dbCollection           *mongo.Collection
	verificationCollection *mongo.Collection
	magicLinkCollection    *mongo.Collection
	mailClient             mailpb.MailServiceClient
	twoFactorService       *TwoFactorService
	storageClient          storagepb.StorageServiceClient
	requestCollection      *mongo.Collection
}

func NewUserServer(
	collection *mongo.Collection,
	verificationCollection *mongo.Collection,
	magicLinkCollection *mongo.Collection,
	mailClient mailpb.MailServiceClient,
	twoFactorService *TwoFactorService,
	storageClient storagepb.StorageServiceClient,
	requestCollection *mongo.Collection,
) *UserServer {
	return &UserServer{
		dbCollection:           collection,
		verificationCollection: verificationCollection,
		magicLinkCollection:    magicLinkCollection,
		mailClient:             mailClient,
		twoFactorService:       twoFactorService,
		storageClient:          storageClient,
		requestCollection:      requestCollection,
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

	if _, err = s.storageClient.CreateFolder(ctx, &storagepb.CreateFolderRequest{Path: verificationToken.UserId}); err != nil {
		slog.Error("verification_failure", "event", "auth.verify.failed", "reason", "silo_error", "status", "failure", "error", err.Error())
	}

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

	jwtToken, err := GenerateToken(user.ID.Hex(), user.Email, user.Role)
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
			StorageQuota: user.StorageQuota,
			UsedQuota:    user.UsedQuota,
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
		StorageQuota:   2000,
		UsedQuota:      0,
		Role:           db.USER,
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
		StorageQuota: newUser.StorageQuota,
		UsedQuota:    0,
		Role:         db.USER.String(),
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
		log.Printf("Database error: %v", err)
		return nil, status.Errorf(codes.Internal, "login lookup failed: %v", err)

	}

	if !user.Verified {
		slog.Warn("login_failure", "event", "auth.login.failed", "reason", "account_not_verified", "user_id", user.ID.Hex(), "status", "failure")
		return nil, status.Error(codes.FailedPrecondition, "your account is not verified. please check your email for the verification link")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		log.Printf("Token generating error: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	if time.Since(user.PasswordActive.Time()) > passwordMaxAge() {
		slog.Warn("login_failure", "event", "auth.login.failed", "reason", "password_expired", "user_id", user.ID.Hex(), "status", "failure")
		return nil, status.Error(codes.FailedPrecondition, "your password has expired; please reset it to continue")
	}

	if user.TwoFactor {
		pendingToken, code, err := s.twoFactorService.StartChallenge(ctx, user.ID.Hex(), user.Email, user.Role)
		if err != nil {
			return nil, err
		}
		_, err = s.mailClient.SendTwoFactor(ctx, &userpb.SendTwoFactorRequest{
			Email: user.Email,
			Code:  code,
			Name:  user.FirstName,
		})
		if err != nil {
			return nil, err
		}

		return &userpb.LoginUserResponse{
			Token:     pendingToken,
			TwoFactor: true,
			User:      nil,
		}, nil
	}

	token, err := GenerateToken(user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		slog.Error("login_failure", "event", "auth.login.failed", "reason", "token_generation_failed", "user_id", user.ID.Hex(), "status", "failure", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "failed to issue token: %v", err)
	}

	slog.Info("login_success", "event", "auth.login", "user_id", user.ID.Hex(), "status", "success")

	return &userpb.LoginUserResponse{
		Token:     token,
		TwoFactor: false,
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
			StorageQuota: user.StorageQuota,
			UsedQuota:    user.UsedQuota,
		},
	}, nil
}

func (s *UserServer) ChangePassword(ctx context.Context, req *userpb.ChangePasswordRequest) (*userpb.ChangePasswordResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	currentPw := req.GetCurrentPassword()
	newPw := req.GetNewPassword()

	if err := validatePassword(newPw); err != nil {
		return nil, err
	}
	if email == "" || currentPw == "" {
		return nil, status.Error(codes.InvalidArgument, "email and current password are required")
	}

	genericResponse := &userpb.ChangePasswordResponse{
		Message: "If an account exists with those credentials and the current password has expired, the new password has been set.",
	}

	var user db.UserDBModel
	err := s.dbCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			_ = bcrypt.CompareHashAndPassword(getDummyHash(), []byte(currentPw))
			slog.Info("change_pw", "event", "auth.change_password", "status", "no_user")
			return genericResponse, nil
		}
		slog.Error("change_pw", "event", "auth.change_password", "status", "db_error", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "lookup failed: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPw)); err != nil {
		slog.Warn("change_pw", "event", "auth.change_password", "status", "bad_password", "user_id", user.ID.Hex())
		return genericResponse, nil
	}

	if time.Since(user.PasswordActive.Time()) <= passwordMaxAge() {
		slog.Warn("change_pw", "event", "auth.change_password", "status", "not_expired", "user_id", user.ID.Hex())
		return genericResponse, nil
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPw), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("change_pw", "event", "auth.change_password", "status", "hash_failed", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "failed to hash new password: %v", err)
	}

	_, err = s.dbCollection.UpdateOne(ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"password":        string(hashed),
			"password_active": primitive.NewDateTimeFromTime(time.Now()),
		}},
	)
	if err != nil {
		slog.Error("change_pw", "event", "auth.change_password", "status", "update_failed", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "failed to update password: %v", err)
	}

	slog.Info("change_pw_success", "event", "auth.change_password", "user_id", user.ID.Hex(), "status", "success")
	return genericResponse, nil
}

func (s *UserServer) Verify2FA(ctx context.Context, req *userpb.Verify2FARequest) (*userpb.Verify2FAResponse, error) {
	session, err := s.twoFactorService.CompleteChallenge(ctx, req.GetPendingToken(), req.GetCode())
	if err != nil {
		slog.Error("verify_2fa", "event", "auth.verify_2fa", "status", "failed_challenge", "error", err.Error())
		return nil, status.Errorf(codes.PermissionDenied, "[user-service] Error while completing challenge:%s", err)
	}

	token, err := GenerateToken(session.UserID, session.Email, session.Role)
	if err != nil {
		slog.Error("verify_2fa", "event", "auth.verify_2fa", "status", "token_gen_failed", "error", err.Error())
		return nil, status.Errorf(codes.PermissionDenied, "[user-service] Error while generating token:%s", err)
	}

	return &userpb.Verify2FAResponse{Token: token}, nil
}

func (s *UserServer) ValidateToken(ctx context.Context, req *userpb.ValidateTokenRequest) (*userpb.ValidateTokenResponse, error) {
	userId, role, err := ParseAndValidateToken(req.GetToken())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid token")
	}

	return &userpb.ValidateTokenResponse{
		UserId: userId,
		Role:   role.String(),
	}, nil
}

func (s *UserServer) CheckStorageQuota(ctx context.Context, req *storagepb.CheckStorageQuotaRequest) (*storagepb.CheckStorageQuotaResponse, error) {
	userId, err := primitive.ObjectIDFromHex(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Value is not a valid id:%v", err)
	}
	var user db.UserDBModel
	err = s.dbCollection.FindOne(ctx, bson.M{"_id": userId}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Errorf(codes.NotFound, "Couldn't find user with ID:%v", err)
		}
		log.Printf("Database error: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to load user from db: %v", err)
	}
	log.Printf("Remaining quota:%d", user.StorageQuota-user.UsedQuota)

	return &storagepb.CheckStorageQuotaResponse{Quota: user.StorageQuota - user.UsedQuota}, nil
}

func (s *UserServer) ChangeUsedQuota(ctx context.Context, req *storagepb.ChangeQuotaRequest) (*userpb.EmptyResponse, error) {
	userId, err := primitive.ObjectIDFromHex(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Value is not a valid id:%v", err)
	}
	res, err := s.dbCollection.UpdateByID(ctx, userId, bson.M{"$inc": bson.M{"used_quota": req.GetAmount()}})
	if err != nil {
		log.Printf("Failed to change quota:%v", err)
		return nil, status.Errorf(codes.Internal, "Failed to change quota:%v", err)
	}
	if res.MatchedCount == 0 {
		return nil, status.Errorf(codes.NotFound, "Couldn't find user with ID: %s", req.GetUserId())
	}
	return &userpb.EmptyResponse{}, nil
}

func (s *UserServer) ListExpansionReq(ctx context.Context, req *userpb.ListExpansionReqRequest) (*userpb.ListExpansionReqResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		fmt.Print("Error loading metadata")
		slog.Error("requests", "event", "quota.list_requests", "status", "no_metadata", "error", "Error loading metadata from request context")
		return nil, status.Error(codes.NotFound, "Error loading metadata")
	}

	tokens := md.Get("authorization")
	if len(tokens) < 1 {
		fmt.Println("No token present")
		slog.Error("requests", "event", "quota.list_requests", "status", "no_token", "error", "No token present in context metadata")
		return nil, status.Error(codes.NotFound, "No token present")
	}

	userIdToken, role, err := ParseAndValidateToken(strings.TrimPrefix(tokens[0], "Bearer "))
	if err != nil {
		slog.Error("requests", "event", "quota.list_requests", "status", "parse_failed", "error", err.Error())
		return nil, status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
	}

	if role != db.ADMIN {
		slog.Error("requests", "event", "quota.list_requests", "status", "unauthorzied", "error", "User is not authorized to do make this action")
		return nil, status.Error(codes.PermissionDenied, "User is not an admin")
	}

	userId, err := primitive.ObjectIDFromHex(userIdToken)
	if err != nil {
		slog.Error("requests", "event", "quota.list_requests", "status", "parse_failed", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "Failed to parse userID as objectID:%v", err)
	}
	var user db.UserDBModel
	err = s.dbCollection.FindOne(ctx, bson.M{"_id": userId}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			request := db.StorageExpansionRequest{ //TEST DELETE AFTER IMPLEMENTING SENDING REQUESTS
				ID:           primitive.NewObjectID(),
				Email:        "random@example.com",
				FirstName:    "Test",
				LastName:     "Test",
				Amount:       3000,
				StorageQuota: 2000,
				UsedQuota:    100,
				Role:         db.USER,
				Status:       db.ON_WAIT,
			}
			s.requestCollection.InsertOne(ctx, request)
			return nil, status.Errorf(codes.NotFound, "Couldn't find user with ID:%v", err)
		}
		log.Printf("Database error: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to load user from db: %v", err)
	}

	if user.Role != db.ADMIN {
		return nil, status.Error(codes.PermissionDenied, "User is not an admin")
	}

	cursor, err := s.requestCollection.Find(ctx, bson.M{"status": db.ON_WAIT})
	if err != nil {
		slog.Error("list_requests", "event", "quota.list_requests", "status", "failed_loading", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "Failed finding all request documents:%v", err)
	}
	defer cursor.Close(ctx)

	var dbRequests []db.StorageExpansionRequest
	if err = cursor.All(ctx, &dbRequests); err != nil {
		slog.Error("list_requests", "event", "quota.list_requests", "status", "failed_parsing", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "Failed to parse documents to requests:%v", err)
	}

	var result []*userpb.StorageExpansionRequest
	for _, r := range dbRequests {
		result = append(result, &userpb.StorageExpansionRequest{
			Email:        r.Email,
			FirstName:    r.FirstName,
			LastName:     r.LastName,
			Amount:       r.Amount,
			StorageQuota: uint64(r.StorageQuota),
			UsedQuota:    uint64(r.UsedQuota),
			Role:         uint32(r.Role),
			Status:       uint32(r.Status),
		})
	}

	if len(result) < 1 {
		request := db.StorageExpansionRequest{ //TEST DELETE AFTER IMPLEMENTING SENDING REQUESTS
			ID:           primitive.NewObjectID(),
			Email:        "random@example.com",
			FirstName:    "Test",
			LastName:     "Test",
			Amount:       3000,
			StorageQuota: 2000,
			UsedQuota:    100,
			Role:         db.USER,
			Status:       db.ON_WAIT,
		}
		s.requestCollection.InsertOne(ctx, request)
	}
	return &userpb.ListExpansionReqResponse{Requests: result}, nil
}

func (s *UserServer) CreateExpansionReq(ctx context.Context, req *userpb.CreateExpansionReqRequest) (*userpb.CreateExpansionReqResponse, error) {
	// Validation
	if req.GetAmount() == 0 {
		return nil, status.Error(codes.InvalidArgument, "Requested amount cannot be zero")
	}

	const maxExpansionMB = 10240
	if req.GetAmount() > maxExpansionMB {
		return nil, status.Errorf(codes.InvalidArgument, "You cannot request more than 10 GB (%d MB) at once", maxExpansionMB)
	}

	// Get token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		slog.Error("create_request", "event", "quota.create_request", "status", "no_metadata")
		return nil, status.Error(codes.Internal, "Error loading metadata")
	}

	tokens := md.Get("authorization")
	if len(tokens) < 1 {
		slog.Error("create_request", "event", "quota.create_request", "status", "no_token")
		return nil, status.Error(codes.Unauthenticated, "No token present")
	}

	// Validate token
	userIdStr, _, err := ParseAndValidateToken(strings.TrimPrefix(tokens[0], "Bearer "))
	if err != nil {
		slog.Error("create_request", "event", "quota.create_request", "status", "invalid_token", "error", err.Error())
		return nil, status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
	}

	userId, err := primitive.ObjectIDFromHex(userIdStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to parse user ID: %v", err)
	}

	// Get user
	var user db.UserDBModel
	err = s.dbCollection.FindOne(ctx, bson.M{"_id": userId}).Decode(&user)
	if err != nil {
		slog.Error("create_request", "event", "quota.create_request", "status", "user_not_found", "error", err.Error())
		return nil, status.Errorf(codes.NotFound, "User not found")
	}

	// Check for previous request
	count, err := s.requestCollection.CountDocuments(ctx, bson.M{"email": user.Email, "status": db.ON_WAIT})
	if err == nil && count > 0 {
		return nil, status.Error(codes.AlreadyExists, "You already have a pending request")
	}

	// Create request
	request := db.StorageExpansionRequest{
		ID:           primitive.NewObjectID(),
		Email:        user.Email,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Amount:       req.GetAmount(),
		StorageQuota: user.StorageQuota,
		UsedQuota:    user.UsedQuota,
		Role:         user.Role,
		Status:       db.ON_WAIT,
	}

	// Save
	_, err = s.requestCollection.InsertOne(ctx, request)
	if err != nil {
		slog.Error("create_request", "event", "quota.create_request", "status", "db_error", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "Failed to save request: %v", err)
	}

	slog.Info("create_request", "event", "quota.create_request", "status", "success", "user_id", userIdStr)

	return &userpb.CreateExpansionReqResponse{
		Message: "Storage expansion request submitted successfully",
	}, nil
}

func (s *UserServer) HandleExpansionReq(ctx context.Context, req *userpb.HandleExpansionReqRequest) (*userpb.HandleExpansionReqResponse, error) {
	// Validate Admin
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "Error loading metadata")
	}

	tokens := md.Get("authorization")
	if len(tokens) < 1 {
		return nil, status.Error(codes.Unauthenticated, "No token present")
	}

	_, role, err := ParseAndValidateToken(strings.TrimPrefix(tokens[0], "Bearer "))
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
	}
	if role != db.ADMIN {
		return nil, status.Error(codes.PermissionDenied, "User is not an admin")
	}

	// Find request
	var expansionReq db.StorageExpansionRequest
	err = s.requestCollection.FindOne(ctx, bson.M{"email": req.GetEmail(), "status": db.ON_WAIT}).Decode(&expansionReq)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Errorf(codes.NotFound, "No pending request found for %s", req.GetEmail())
		}
		return nil, status.Errorf(codes.Internal, "Database error: %v", err)
	}

	// Update if approved
	if req.GetApprove() {
		_, err = s.dbCollection.UpdateOne(
			ctx,
			bson.M{"email": req.GetEmail()},
			bson.M{"$inc": bson.M{"storage_quota": int32(expansionReq.Amount)}},
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to update user quota: %v", err)
		}
	}

	// Update request status
	newStatus := db.REJECTED
	if req.GetApprove() {
		newStatus = db.ACCEPTED
	}

	_, err = s.requestCollection.UpdateOne(
		ctx,
		bson.M{"_id": expansionReq.ID},
		bson.M{"$set": bson.M{"status": newStatus}},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update request status: %v", err)
	}

	action := "rejected"
	if req.GetApprove() {
		action = "approved"
	}

	return &userpb.HandleExpansionReqResponse{
		Message: fmt.Sprintf("Request for %s successfully %s", req.GetEmail(), action),
	}, nil
}

func (s *UserServer) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	// 1. Validate Admin via Metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "Error loading metadata")
	}

	tokens := md.Get("authorization")
	if len(tokens) < 1 {
		return nil, status.Error(codes.Unauthenticated, "No token present")
	}

	_, role, err := ParseAndValidateToken(strings.TrimPrefix(tokens[0], "Bearer "))
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
	}

	if role != db.ADMIN {
		return nil, status.Error(codes.PermissionDenied, "User is not an admin")
	}

	// 2. Fetch all users from MongoDB
	cursor, err := s.dbCollection.Find(ctx, bson.D{})
	if err != nil {
		slog.Error("list_users", "event", "admin.list_users", "status", "failed_loading", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "Failed finding users: %v", err)
	}
	defer cursor.Close(ctx)

	var dbUsers []db.UserDBModel
	if err = cursor.All(ctx, &dbUsers); err != nil {
		slog.Error("list_users", "event", "admin.list_users", "status", "failed_parsing", "error", err.Error())
		return nil, status.Errorf(codes.Internal, "Failed to parse users: %v", err)
	}

	// 3. Map DB structs to Protobuf structs
	var result []*userpb.UserProfile
	for _, u := range dbUsers {
		result = append(result, &userpb.UserProfile{
			Id:           u.ID.Hex(),
			Email:        u.Email,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			StorageQuota: u.StorageQuota,
			UsedQuota:    u.UsedQuota,
			Role:         u.Role.String(),
		})
	}

	return &userpb.ListUsersResponse{Users: result}, nil
}
