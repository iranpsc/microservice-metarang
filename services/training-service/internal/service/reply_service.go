package service

import (
	"context"
	"fmt"

	"metargb/training-service/internal/models"
	"metargb/training-service/internal/repository"
)

type ReplyService struct {
	commentRepo *repository.CommentRepository
	userRepo    *repository.UserRepository
}

func NewReplyService(commentRepo *repository.CommentRepository, userRepo *repository.UserRepository) *ReplyService {
	return &ReplyService{
		commentRepo: commentRepo,
		userRepo:    userRepo,
	}
}

// GetReplies retrieves replies for a comment
func (s *ReplyService) GetReplies(ctx context.Context, commentID uint64, page, perPage int32) ([]*CommentDetails, int32, error) {
	replies, total, err := s.commentRepo.GetReplies(ctx, commentID, page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get replies: %w", err)
	}

	details := make([]*CommentDetails, 0, len(replies))
	for _, reply := range replies {
		detail, err := s.getReplyDetails(ctx, reply)
		if err != nil {
			continue // Skip replies with errors
		}
		details = append(details, detail)
	}

	return details, total, nil
}

// AddReply creates a new reply to a comment
func (s *ReplyService) AddReply(ctx context.Context, parentCommentID, userID uint64, content string) (*CommentDetails, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > 2000 {
		return nil, fmt.Errorf("content must be at most 2000 characters")
	}

	// Verify parent comment exists and user is not the author
	parentComment, err := s.commentRepo.GetCommentByID(ctx, parentCommentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent comment: %w", err)
	}
	if parentComment == nil {
		return nil, fmt.Errorf("parent comment not found")
	}
	if parentComment.UserID == userID {
		return nil, fmt.Errorf("user cannot reply to their own comment")
	}

	reply, err := s.commentRepo.AddReply(ctx, parentCommentID, userID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to add reply: %w", err)
	}

	return s.getReplyDetails(ctx, reply)
}

// UpdateReply updates a reply
func (s *ReplyService) UpdateReply(ctx context.Context, replyID, userID uint64, content string) (*CommentDetails, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > 2000 {
		return nil, fmt.Errorf("content must be at most 2000 characters")
	}

	// Verify ownership
	reply, err := s.commentRepo.GetCommentByID(ctx, replyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reply: %w", err)
	}
	if reply == nil {
		return nil, fmt.Errorf("reply not found")
	}
	if reply.UserID != userID {
		return nil, fmt.Errorf("user not authorized to update this reply")
	}

	if err := s.commentRepo.UpdateReply(ctx, replyID, userID, content); err != nil {
		return nil, fmt.Errorf("failed to update reply: %w", err)
	}

	// Get updated reply
	updatedReply, err := s.commentRepo.GetCommentByID(ctx, replyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated reply: %w", err)
	}

	return s.getReplyDetails(ctx, updatedReply)
}

// DeleteReply deletes a reply
func (s *ReplyService) DeleteReply(ctx context.Context, replyID, userID uint64) error {
	return s.commentRepo.DeleteReply(ctx, replyID, userID)
}

// AddReplyInteraction adds or updates a user's interaction on a reply
func (s *ReplyService) AddReplyInteraction(ctx context.Context, replyID, userID uint64, liked bool, ipAddress string) error {
	// Verify user is not the reply author
	reply, err := s.commentRepo.GetCommentByID(ctx, replyID)
	if err != nil {
		return fmt.Errorf("failed to get reply: %w", err)
	}
	if reply == nil {
		return fmt.Errorf("reply not found")
	}
	if reply.UserID == userID {
		return fmt.Errorf("user cannot react to their own reply")
	}

	return s.commentRepo.AddReplyInteraction(ctx, replyID, userID, liked, ipAddress)
}

// getReplyDetails enriches a reply with user info and stats
func (s *ReplyService) getReplyDetails(ctx context.Context, reply *models.Comment) (*CommentDetails, error) {
	// Reuse the same logic as comment details
	commentService := NewCommentService(s.commentRepo, s.userRepo)
	return commentService.getCommentDetails(ctx, reply)
}
