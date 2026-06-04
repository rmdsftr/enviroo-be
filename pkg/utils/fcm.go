package utils 

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMClient wraps Firebase Cloud Messaging.
type FCMClient struct {
	client *messaging.Client
}

// NewFCMClient initialises FCM using a service-account JSON key file.
// credentialsFile is the path to your Firebase service-account JSON.
func NewFCMClient(ctx context.Context, credentialsFile string) (*FCMClient, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi fcm client: %w", err)
	}

	return &FCMClient{client: client}, nil
}

// FCMPayload holds the data needed to send a push notification.
type FCMPayload struct {
	Token string // device FCM token

	Title string
	Body  string

	// Data is an optional key-value map sent alongside the notification.
	// Useful for deep-linking in Flutter (e.g. ref_id, ref_type).
	Data map[string]string
}

// SendToDevice sends a notification to a single device token.
// Returns the FCM message ID on success.
func (f *FCMClient) SendToDevice(ctx context.Context, payload FCMPayload) (string, error) {
	if payload.Token == "" {
		return "", fmt.Errorf("fcm token kosong")
	}

	msg := &messaging.Message{
		Token: payload.Token,
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Sound: "default",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "default",
				},
			},
		},
	}

	if len(payload.Data) > 0 {
		msg.Data = payload.Data
	}

	messageID, err := f.client.Send(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("gagal mengirim fcm: %w", err)
	}

	log.Printf("[FCM] Berhasil kirim notifikasi ke token %s — ID: %s", maskToken(payload.Token), messageID)
	return messageID, nil
}

// SendToMultipleDevices sends the same notification to up to 500 tokens at once.
func (f *FCMClient) SendToMultipleDevices(ctx context.Context, tokens []string, payload FCMPayload) (*messaging.BatchResponse, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("daftar token kosong")
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Sound: "default",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "default",
				},
			},
		},
	}

	if len(payload.Data) > 0 {
		msg.Data = payload.Data
	}

	resp, err := f.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim fcm multicast: %w", err)
	}

	log.Printf("[FCM] Multicast selesai — sukses: %d, gagal: %d", resp.SuccessCount, resp.FailureCount)
	return resp, nil
}

// maskToken menyembunyikan sebagian token untuk log aman.
func maskToken(token string) string {
	if len(token) <= 10 {
		return "***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}