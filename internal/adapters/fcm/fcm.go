package fcm

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type Client struct {
	client *messaging.Client
}

func New(ctx context.Context, credentialsFile string) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi fcm client: %w", err)
	}

	return &Client{client: client}, nil
}

type Payload struct {
	Token string

	Title string
	Body  string

	Data map[string]string
}

func (f *Client) SendToDevice(ctx context.Context, payload Payload) (string, error) {
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

func (f *Client) SendToMultipleDevices(ctx context.Context, tokens []string, payload Payload) (*messaging.BatchResponse, error) {
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

func maskToken(token string) string {
	if len(token) <= 10 {
		return "***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}
