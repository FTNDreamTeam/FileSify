package repo

import (
	"context"
	"time"

	notificationpb "gen/notification-service-gen"

	"github.com/gocql/gocql"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DI
type NotificationRepo struct {
	session *gocql.Session
}

func NewNotificationRepo(session *gocql.Session) *NotificationRepo {
	return &NotificationRepo{session: session}
}

func (r *NotificationRepo) List(ctx context.Context, userID string) ([]*notificationpb.Notification, error) {
	// Query
	query := `SELECT notification_id, type, channel, title, body, is_read, actor_user_id, resource_id, resource_type, created_at 
	          FROM notifications 
	          WHERE user_id = ? 
	          LIMIT 50`

	// Executing and storing response
	iter := r.session.Query(query, userID).WithContext(ctx).Iter()

	// Variable declaration
	var notifications []*notificationpb.Notification
	var scanID gocql.UUID
	var scanType, scanChannel, scanTitle, scanBody, scanActorID, scanResourceID, scanResourceType string
	var scanIsRead bool
	var scanCreatedAt time.Time

	// Scan to get individual notifications
	for iter.Scan(&scanID, &scanType, &scanChannel, &scanTitle, &scanBody, &scanIsRead, &scanActorID, &scanResourceID, &scanResourceType, &scanCreatedAt) {
		notifications = append(notifications, &notificationpb.Notification{
			Id:           scanID.String(),
			Type:         scanType,
			Channel:      scanChannel,
			Title:        scanTitle,
			Body:         scanBody,
			IsRead:       scanIsRead,
			ActorUserId:  scanActorID,
			ResourceId:   scanResourceID,
			ResourceType: scanResourceType,
			CreatedAt:    timestamppb.New(scanCreatedAt),
		})
	}

	// Error check + return
	if err := iter.Close(); err != nil {
		return nil, err
	}

	return notifications, nil
}
