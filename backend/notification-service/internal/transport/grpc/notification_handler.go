package grpc

import (
	"context"
	"errors"

	notificationpb "gen/notification-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/notification-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NotificationHandler struct {
	notificationpb.UnimplementedNotificationServiceServer
	service *service.NotificationService
}

func NewNotificationHandler(s *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: s}
}

func (h *NotificationHandler) ListNotifications(ctx context.Context, req *notificationpb.ListNotificationsRequest) (*notificationpb.ListNotificationsResponse, error) {
	notifications, err := h.service.ListNotifications(ctx, req.GetToken())
	if err != nil {
		// Error
		if errors.Is(err, service.ErrTokenRequired) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrInvalidToken) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to fetch notifications: %v", err)
	}

	// Return notifications
	return &notificationpb.ListNotificationsResponse{
		Notifications: notifications,
	}, nil
}
