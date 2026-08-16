package client

import (
	"context"
	"fmt"
	"time"

	pb "metarang/shared/pb/notifications"
	grpcutil "metarang/shared/pkg/grpc"

	"google.golang.org/grpc"
)

// NotificationClient wraps gRPC client for Notification Service
type NotificationClient struct {
	client pb.NotificationServiceClient
	conn   *grpc.ClientConn
}

// NewNotificationClient creates a new Notification Service client
func NewNotificationClient(address string) (*NotificationClient, error) {
	conn, err := grpcutil.DialContextWithTimeout(address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to notification service at %s: %w", address, err)
	}

	return &NotificationClient{
		client: pb.NewNotificationServiceClient(conn),
		conn:   conn,
	}, nil
}

// NewNotificationClientFromGRPC builds a NotificationClient from an existing gRPC stub (tests).
func NewNotificationClientFromGRPC(grpcClient pb.NotificationServiceClient) *NotificationClient {
	return &NotificationClient{client: grpcClient}
}

// Close closes the gRPC connection
func (c *NotificationClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendNotification sends a notification to a user
func (c *NotificationClient) SendNotification(ctx context.Context, userID uint64, notificationType, title, message string, data map[string]string) error {
	req := &pb.SendNotificationRequest{
		UserId:    userID,
		Type:      notificationType,
		Title:     title,
		Message:   message,
		Data:      data,
		SendSms:   false,
		SendEmail: false,
	}

	_, err := c.client.SendNotification(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

// SendBuyRequestNotification sends a buy request notification to buyer or seller
// type must be "buyer" or "seller"
func (c *NotificationClient) SendBuyRequestNotification(ctx context.Context, userID uint64, notificationType string, buyRequestID, featureID uint64, pricePSC, priceIRR float64) error {
	var title, message string
	data := map[string]string{
		"buy_request_id": fmt.Sprintf("%d", buyRequestID),
		"feature_id":     fmt.Sprintf("%d", featureID),
		"price_psc":      fmt.Sprintf("%.0f", pricePSC),
		"price_irr":      fmt.Sprintf("%.0f", priceIRR),
		"type":           notificationType,
	}

	if notificationType == "buyer" {
		title = "درخواست خرید ارسال شد"
		message = fmt.Sprintf("مبلغ %.0f psc و %.0f از حساب شما بابت پیشنهاد خرید ملک %d برداشت شد.", pricePSC, priceIRR, featureID)
		data["related-to"] = "transactions"
	} else {
		title = "درخواست خرید دریافت شد"
		message = fmt.Sprintf("یک پیشنهاد خرید برای ملک %d دریافت شد.", featureID)
		data["related-to"] = "transactions"
	}

	return c.SendNotification(ctx, userID, "BuyRequestNotification", title, message, data)
}

// SendBuyFeatureNotification sends a notification when a feature is purchased
// Different messages for RGB purchases (color) vs user-to-user (PSC+IRR)
func (c *NotificationClient) SendBuyFeatureNotification(ctx context.Context, userID uint64, featureID uint64, isRGBPurchase bool, color string, stability float64, pscAmount, irrAmount float64) error {
	var title, message string
	data := map[string]string{
		"feature_id": fmt.Sprintf("%d", featureID),
		"related-to": "transactions",
	}

	title = "خریداری ملک"

	if isRGBPurchase {
		message = fmt.Sprintf("%.2f لیتر رنگ %s از حساب شما بابت خرید زمین %d برداشت شد.", stability, color, featureID)
		data["stability"] = fmt.Sprintf("%.2f", stability)
		data["color"] = color
		data["purchase_type"] = "rgb"
	} else {
		message = fmt.Sprintf("از حساب شما %.0f psc و %.0f ریال بابت خرید ملک %d برداشت شد.", pscAmount, irrAmount, featureID)
		data["psc_amount"] = fmt.Sprintf("%.0f", pscAmount)
		data["irr_amount"] = fmt.Sprintf("%.0f", irrAmount)
		data["purchase_type"] = "user"
	}

	return c.SendNotification(ctx, userID, "BuyFeatureNotification", title, message, data)
}

// SendSellRequestNotification sends a notification when a sell request is created
func (c *NotificationClient) SendSellRequestNotification(ctx context.Context, sellerID uint64, featureID uint64, featurePropertiesID string) error {
	title := "درخواست فروش ملک"
	message := fmt.Sprintf("ملک %s با موفقیت قیمت گذاری شد.", featurePropertiesID)
	data := map[string]string{
		"feature_id":    fmt.Sprintf("%d", featureID),
		"properties_id": featurePropertiesID,
		"related-to":    "sell-requests",
	}

	return c.SendNotification(ctx, sellerID, "SellRequestNotification", title, message, data)
}

// SendFeatureHourlyProfitDeposit sends a notification when hourly profit is withdrawn
func (c *NotificationClient) SendFeatureHourlyProfitDeposit(ctx context.Context, userID uint64, asset string, amount float64, karbari string, featurePropertiesID string) error {
	// Get color name in Persian
	var colorName string
	switch asset {
	case "yellow":
		colorName = "زرد"
	case "red":
		colorName = "قرمز"
	case "blue":
		colorName = "آبی"
	default:
		colorName = asset
	}

	// Get karbari title in Persian
	var karbariTitle string
	switch karbari {
	case "m":
		karbariTitle = "مسکونی"
	case "t":
		karbariTitle = "تجاری"
	case "a":
		karbariTitle = "آموزشی"
	default:
		karbariTitle = karbari
	}

	title := fmt.Sprintf("سود ساعتی %s", karbariTitle)
	message := fmt.Sprintf("مبلغ %.6f %s به کیف پول شما اضافه شد", amount, colorName)
	data := map[string]string{
		"asset":   asset,
		"amount":  fmt.Sprintf("%.6f", amount),
		"karbari": karbariTitle,
	}

	if featurePropertiesID != "" {
		data["id"] = featurePropertiesID
	}

	return c.SendNotification(ctx, userID, "FeatureHourlyProfitDeposit", title, message, data)
}
