package service

import (
	"context"
	"fmt"
	"metargb/support-service/internal/models"
	"metargb/support-service/internal/repository"
)

type NoteService interface {
	CreateNote(ctx context.Context, userID uint64, title, content, attachment string) (*models.Note, error)
	GetNotes(ctx context.Context, userID uint64) ([]*models.Note, error)
	GetNote(ctx context.Context, noteID, userID uint64) (*models.Note, error)
	UpdateNote(ctx context.Context, noteID, userID uint64, title, content, attachment string) (*models.Note, error)
	DeleteNote(ctx context.Context, noteID, userID uint64) error
}

type noteService struct {
	noteRepo repository.NoteRepository
}

func NewNoteService(noteRepo repository.NoteRepository) NoteService {
	return &noteService{
		noteRepo: noteRepo,
	}
}

func (s *noteService) CreateNote(ctx context.Context, userID uint64, title, content, attachment string) (*models.Note, error) {
	note := &models.Note{
		Title:      title,
		Content:    content,
		Attachment: attachment,
		UserID:     userID,
	}

	return s.noteRepo.Create(ctx, note)
}

func (s *noteService) GetNotes(ctx context.Context, userID uint64) ([]*models.Note, error) {
	return s.noteRepo.GetByUserID(ctx, userID)
}

func (s *noteService) GetNote(ctx context.Context, noteID, userID uint64) (*models.Note, error) {
	// Check authorization - only owner can view
	owned, err := s.noteRepo.CheckUserOwnership(ctx, noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check ownership: %w", err)
	}
	if !owned {
		return nil, fmt.Errorf("unauthorized: you don't have permission to view this note")
	}

	return s.noteRepo.GetByID(ctx, noteID)
}

func (s *noteService) UpdateNote(ctx context.Context, noteID, userID uint64, title, content, attachment string) (*models.Note, error) {
	// Check authorization - only owner can update
	owned, err := s.noteRepo.CheckUserOwnership(ctx, noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check ownership: %w", err)
	}
	if !owned {
		return nil, fmt.Errorf("unauthorized: you don't have permission to update this note")
	}

	// Get existing note
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}
	if note == nil {
		return nil, fmt.Errorf("note not found")
	}

	// Update fields
	note.Title = title
	note.Content = content
	note.Attachment = attachment

	err = s.noteRepo.Update(ctx, note)
	if err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	return note, nil
}

func (s *noteService) DeleteNote(ctx context.Context, noteID, userID uint64) error {
	// Check authorization - only owner can delete
	owned, err := s.noteRepo.CheckUserOwnership(ctx, noteID, userID)
	if err != nil {
		return fmt.Errorf("failed to check ownership: %w", err)
	}
	if !owned {
		return fmt.Errorf("unauthorized: you don't have permission to delete this note")
	}

	return s.noteRepo.Delete(ctx, noteID)
}
