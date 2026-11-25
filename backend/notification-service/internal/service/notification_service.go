package service

import (
	"context"
	"errors"
	"log"

	notificationpb "gen/notification-service-gen"
	userpb "gen/user-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/notification-service/internal/repo"
)

var (
	ErrTokenRequired = errors.New("token is required")
	ErrInvalidToken  = errors.New("invalid or expired token")
)

type NotificationService struct {
	repo       *repo.NotificationRepo
	userClient userpb.UserServiceClient
}

func NewNotificationService(r *repo.NotificationRepo, uc userpb.UserServiceClient) *NotificationService {
	return &NotificationService{repo: r, userClient: uc}
}

func (s *NotificationService) ListNotifications(ctx context.Context, token string) ([]*notificationpb.Notification, error) {
	if token == "" {
		return nil, ErrTokenRequired
	}

	// Validate JWT with User Service
	valResp, err := s.userClient.ValidateToken(ctx, &userpb.ValidateTokenRequest{Token: token})
	if err != nil {
		log.Printf("User Service rejected token: %v", err)
		return nil, ErrInvalidToken
	}

	userID := valResp.GetUserId()
	if userID == "" {
		return nil, ErrInvalidToken
	}

	// Call repo + return notifications
	return s.repo.List(ctx, userID)
}
