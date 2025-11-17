package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// GetDynastyMessage retrieves a dynasty message template by type
func (r *MessageRepository) GetDynastyMessage(ctx context.Context, messageType string) (string, error) {
	query := `SELECT message FROM dynasty_messages WHERE type = ? LIMIT 1`
	
	var message string
	err := r.db.QueryRowContext(ctx, query, messageType).Scan(&message)
	if err == sql.ErrNoRows {
		return "", nil // Return empty string if not found
	}
	if err != nil {
		return "", fmt.Errorf("failed to get dynasty message: %w", err)
	}
	
	return message, nil
}

// FormatMessageWithPlaceholders replaces placeholders in message template
func (r *MessageRepository) FormatMessageWithPlaceholders(template string, replacements map[string]string) string {
	result := template
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// GetRelationshipTitle returns Persian title for relationship
func (r *MessageRepository) GetRelationshipTitle(relationship string) string {
	titles := map[string]string{
		"brother":   "برادر",
		"sister":    "خواهر",
		"offspring": "فرزند",
		"father":    "پدر",
		"mother":    "مادر",
		"husband":   "شوهر",
		"wife":      "زن",
		"owner":     "مالک",
	}
	
	if title, ok := titles[relationship]; ok {
		return title
	}
	return relationship
}

// PrepareJoinRequestMessages prepares both sender and receiver messages
func (r *MessageRepository) PrepareJoinRequestMessages(ctx context.Context, senderCode, receiverCode, senderName, receiverName, relationship, date string) (senderMsg string, receiverMsg string, err error) {
	// Get message templates
	senderTemplate, err := r.GetDynastyMessage(ctx, "requester_confirmation_message")
	if err != nil {
		return "", "", fmt.Errorf("failed to get sender template: %w", err)
	}
	
	receiverTemplate, err := r.GetDynastyMessage(ctx, "reciever_message")
	if err != nil {
		return "", "", fmt.Errorf("failed to get receiver template: %w", err)
	}
	
	// Get relationship title in Persian
	relationshipTitle := r.GetRelationshipTitle(relationship)
	
	// Prepare replacements
	replacements := map[string]string{
		"[sender-code]":    senderCode,
		"[reciever-code]":  receiverCode,
		"[sender-name]":    senderName,
		"[reciever-name]":  receiverName,
		"[relationship]":   relationshipTitle,
		"[created_at]":     date,
	}
	
	// Format messages
	senderMsg = r.FormatMessageWithPlaceholders(senderTemplate, replacements)
	receiverMsg = r.FormatMessageWithPlaceholders(receiverTemplate, replacements)
	
	return senderMsg, receiverMsg, nil
}

// PrepareAcceptMessages prepares messages for accepting join request
func (r *MessageRepository) PrepareAcceptMessages(ctx context.Context, requesterCode, receiverCode, requesterName, receiverName, relationship, date string) (requesterMsg string, receiverMsg string, err error) {
	// Get message templates
	requesterTemplate, err := r.GetDynastyMessage(ctx, "requester_accept_message")
	if err != nil {
		return "", "", fmt.Errorf("failed to get requester accept template: %w", err)
	}
	
	receiverTemplate, err := r.GetDynastyMessage(ctx, "reciever_accept_message")
	if err != nil {
		return "", "", fmt.Errorf("failed to get receiver accept template: %w", err)
	}
	
	// Get relationship title
	relationshipTitle := r.GetRelationshipTitle(relationship)
	
	// Prepare replacements
	replacements := map[string]string{
		"[sender-code]":    requesterCode,
		"[reciever-code]":  receiverCode,
		"[sender-name]":    requesterName,
		"[reciever-name]":  receiverName,
		"[relationship]":   relationshipTitle,
		"[created_at]":     date,
	}
	
	// Format messages
	requesterMsg = r.FormatMessageWithPlaceholders(requesterTemplate, replacements)
	receiverMsg = r.FormatMessageWithPlaceholders(receiverTemplate, replacements)
	
	return requesterMsg, receiverMsg, nil
}

// PrepareRejectMessages prepares messages for rejecting join request
func (r *MessageRepository) PrepareRejectMessages(ctx context.Context, requesterCode, receiverCode string) (requesterMsg string, receiverMsg string, err error) {
	// For rejection, Laravel uses hardcoded messages, not templates
	requesterMsg = fmt.Sprintf("درخواست پیوستن به سلسله شما توسط کاربر %s رد شد!", receiverCode)
	receiverMsg = fmt.Sprintf("درخواست پیوستن به سلسله از طرف %s توسط شما رد شد.", requesterCode)
	
	return requesterMsg, receiverMsg, nil
}

