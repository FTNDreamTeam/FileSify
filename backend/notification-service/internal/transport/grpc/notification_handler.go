package grpc

import (
	"context"
	"errors"

	notificationpb "gen/notification-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/notification-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DI
type NotificationHandler struct {
	notificationpb.UnimplementedNotificationServiceServer
	service *service.NotificationService
}

func NewNotificationHandler(s *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: s}
}

func (h *NotificationHandler) ListNotifications(ctx context.Context, req *notificationpb.ListNotificationsRequest) (*notificationpb.ListNotificationsResponse, error) {
	// Call service
	notifications, err := h.service.ListNotifications(ctx, req.Token)

	// Error
	if err != nil {
		if errors.Is(err, service.ErrUserIDRequired) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to fetch notifications: %v", err)
	}

	// Response return
	return &notificationpb.ListNotificationsResponse{
		Notifications: notifications,
	}, nil
}
