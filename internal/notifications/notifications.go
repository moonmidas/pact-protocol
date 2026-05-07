package notifications

import (
	"fmt"
	"time"

	"pact/internal/protocol"
	"pact/internal/store"
)

const (
	TypeApprovalNeeded = "pact.approval.needed"
	TypeDecisionSaved  = "pact.decision.saved"
)

func EnsureForRequests(paths store.Paths, profile string) ([]protocol.NotificationRecord, error) {
	requests, err := store.ReadJSONArray[protocol.RequestRecord](paths.RequestsFile(profile))
	if err != nil {
		return nil, err
	}
	notifications, err := store.ReadJSONArray[protocol.NotificationRecord](paths.NotificationsFile(profile))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for _, request := range requests {
		if request.State != protocol.StateDelivered {
			continue
		}
		if hasNotification(notifications, TypeApprovalNeeded, request.ID) {
			continue
		}
		id, err := protocol.NewID("note")
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, protocol.NotificationRecord{
			ID:        id,
			Type:      TypeApprovalNeeded,
			Source:    "pact://local/" + profile,
			Subject:   "pact://request/" + request.ID,
			Profile:   profile,
			Title:     fmt.Sprintf("%s asks for %s", request.FromProfile, request.Artifact.Name),
			Body:      fmt.Sprintf("Review, approve, counter, or reject %s from %s.", request.Artifact.Name, request.FromProfile),
			RequestID: request.ID,
			State:     "unread",
			Read:      false,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if err := store.WriteJSONArray(paths.NotificationsFile(profile), notifications); err != nil {
		return nil, err
	}
	return notifications, nil
}

func List(paths store.Paths, profile string, unreadOnly bool) ([]protocol.NotificationRecord, error) {
	notifications, err := EnsureForRequests(paths, profile)
	if err != nil {
		return nil, err
	}
	if !unreadOnly {
		return notifications, nil
	}
	unread := make([]protocol.NotificationRecord, 0)
	for _, notification := range notifications {
		if !notification.Read {
			unread = append(unread, notification)
		}
	}
	return unread, nil
}

func MarkRead(paths store.Paths, profile string, id string) (protocol.NotificationRecord, error) {
	notifications, err := EnsureForRequests(paths, profile)
	if err != nil {
		return protocol.NotificationRecord{}, err
	}
	for i := range notifications {
		if notifications[i].ID == id {
			notifications[i].Read = true
			notifications[i].State = "read"
			notifications[i].UpdatedAt = time.Now().UTC()
			if err := store.WriteJSONArray(paths.NotificationsFile(profile), notifications); err != nil {
				return protocol.NotificationRecord{}, err
			}
			return notifications[i], nil
		}
	}
	return protocol.NotificationRecord{}, fmt.Errorf("notification not found: %s", id)
}

func hasNotification(notifications []protocol.NotificationRecord, kind string, requestID string) bool {
	for _, notification := range notifications {
		if notification.Type == kind && notification.RequestID == requestID {
			return true
		}
	}
	return false
}
