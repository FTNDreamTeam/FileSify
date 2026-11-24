package service

import (
	"context"
	"errors"

	notificationpb "gen/notification-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/notification-service/internal/repo"
)

// Error
var (
	ErrUserIDRequired = errors.New("user_id (token field) is required")
)

// DI
type NotificationService struct {
	repo *repo.NotificationRepo
}

func NewNotificationService(r *repo.NotificationRepo) *NotificationService {
	return &NotificationService{repo: r}
}

func (s *NotificationService) ListNotifications(ctx context.Context, userID string) ([]*notificationpb.Notification, error) {
	// Error
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	// Call repo + return
	return s.repo.List(ctx, userID)
}
