package services

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

var fcmClient *messaging.Client

// InitFCM initializes Firebase Cloud Messaging
func InitFCM() error {
	ctx := context.Background()

	// Initialize Firebase app
	opt := option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return fmt.Errorf("failed to initialize Firebase app: %v", err)
	}

	// Initialize FCM client
	client, err := app.Messaging(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize FCM client: %v", err)
	}

	fcmClient = client
	return nil
}

// SendPushNotification sends a notification to a single device
func SendPushNotification(deviceToken, title, body string, data map[string]string) (string, error) {
	if fcmClient == nil {
		return "", fmt.Errorf("FCM client not initialized")
	}

	ctx := context.Background()

	message := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	response, err := fcmClient.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to send notification: %v", err)
	}

	return response, nil
}

// SendBulkPushNotifications sends notifications to multiple devices
func SendBulkPushNotifications(deviceTokens []string, title, body string, data map[string]string) ([]string, error) {
	if fcmClient == nil {
		return nil, fmt.Errorf("FCM client not initialized")
	}

	ctx := context.Background()

	message := &messaging.MulticastMessage{
		Tokens: deviceTokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	response, err := fcmClient.SendMulticast(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send bulk notifications: %v", err)
	}

	// Return successful message IDs
	results := []string{}
	for i, resp := range response.Responses {
		if resp.Success {
			results = append(results, deviceTokens[i])
		}
	}

	return results, nil
}

// GetServiceStatus returns FCM service connection status
func GetServiceStatus() map[string]interface{} {
	status := map[string]interface{}{
		"initialized": fcmClient != nil,
		"service":     "Firebase Cloud Messaging",
	}

	if fcmClient != nil {
		status["status"] = "connected"
	} else {
		status["status"] = "not initialized"
	}

	return status
}
