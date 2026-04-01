package events

// NotificationEvent represents a normalized event for the notification system.
// Published by the Connect interceptor, consumed by the notification consumer.
type NotificationEvent struct {
	Type      string         `json:"type"` // stage_change, comment, mention, approve, reject
	ActorID   int64          `json:"actor_id"`
	ChangeID  int64          `json:"change_id"`
	ProjectID int64          `json:"project_id"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
