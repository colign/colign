package push

import (
	"context"
	"encoding/json"
	"log/slog"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
)

type Service struct {
	db             *bun.DB
	vapidPublicKey string
	vapidPrivate   string
	vapidSubject   string
}

func NewService(db *bun.DB, vapidPublicKey, vapidPrivate, vapidSubject string) *Service {
	return &Service{
		db:             db,
		vapidPublicKey: vapidPublicKey,
		vapidPrivate:   vapidPrivate,
		vapidSubject:   vapidSubject,
	}
}

// VAPIDPublicKey returns the public key for client-side subscription.
func (s *Service) VAPIDPublicKey() string {
	return s.vapidPublicKey
}

// Subscribe registers or updates a push subscription for a user.
func (s *Service) Subscribe(ctx context.Context, userID int64, endpoint, p256dh, auth string) error {
	sub := &models.PushSubscription{
		UserID:   userID,
		Endpoint: endpoint,
		P256dh:   p256dh,
		Auth:     auth,
	}
	_, err := s.db.NewInsert().Model(sub).
		On("CONFLICT (user_id, endpoint) DO UPDATE").
		Set("p256dh = EXCLUDED.p256dh").
		Set("auth = EXCLUDED.auth").
		Exec(ctx)
	return err
}

// Unsubscribe removes a push subscription.
func (s *Service) Unsubscribe(ctx context.Context, userID int64, endpoint string) error {
	_, err := s.db.NewDelete().Model((*models.PushSubscription)(nil)).
		Where("user_id = ?", userID).
		Where("endpoint = ?", endpoint).
		Exec(ctx)
	return err
}

// Payload is the JSON structure sent to the browser.
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
	Icon  string `json:"icon,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

// SendToUser sends a push notification to all subscriptions of a user.
func (s *Service) SendToUser(ctx context.Context, userID int64, payload Payload) {
	if s.vapidPublicKey == "" {
		return
	}

	var subs []models.PushSubscription
	if err := s.db.NewSelect().Model(&subs).
		Where("user_id = ?", userID).
		Scan(ctx); err != nil {
		slog.WarnContext(ctx, "push: failed to list subscriptions", slog.String("error", err.Error()))
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, sub := range subs {
		wpSub := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification(data, wpSub, &webpush.Options{
			VAPIDPublicKey:  s.vapidPublicKey,
			VAPIDPrivateKey: s.vapidPrivate,
			Subscriber:      s.vapidSubject,
		})
		if err != nil {
			slog.WarnContext(ctx, "push: send failed", slog.String("endpoint", sub.Endpoint), slog.String("error", err.Error()))
			continue
		}
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "push: close response body failed", slog.String("error", err.Error()))
		}

		// Remove expired/invalid subscriptions
		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			if err := s.Unsubscribe(ctx, userID, sub.Endpoint); err != nil {
				slog.WarnContext(ctx, "push: cleanup failed", slog.String("error", err.Error()))
			}
		}
	}
}
