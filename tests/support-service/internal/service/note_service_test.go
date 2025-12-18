package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"metargb/support-service/internal/models"
)

// mockNoteRepository implements NoteRepository for testing
type mockNoteRepository struct {
	notes           map[uint64]*models.Note
	userNotes       map[uint64][]*models.Note
	createCount     int
	updateCount     int
	deleteCount     int
	getByIDFunc     func(ctx context.Context, noteID uint64) (*models.Note, error)
	getByUserIDFunc func(ctx context.Context, userID uint64) ([]*models.Note, error)
}

func newMockNoteRepository() *mockNoteRepository {
	return &mockNoteRepository{
		notes:     make(map[uint64]*models.Note),
		userNotes: make(map[uint64][]*models.Note),
	}
}

func (m *mockNoteRepository) Create(ctx context.Context, note *models.Note) (*models.Note, error) {
	m.createCount++
	id := uint64(len(m.notes) + 1)
	note.ID = id
	note.CreatedAt = time.Now()
	note.UpdatedAt = time.Now()
	m.notes[id] = note
	m.userNotes[note.UserID] = append(m.userNotes[note.UserID], note)
	return note, nil
}

func (m *mockNoteRepository) GetByID(ctx context.Context, noteID uint64) (*models.Note, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, noteID)
	}
	return m.notes[noteID], nil
}

func (m *mockNoteRepository) GetByUserID(ctx context.Context, userID uint64) ([]*models.Note, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return m.userNotes[userID], nil
}

func (m *mockNoteRepository) Update(ctx context.Context, note *models.Note) error {
	m.updateCount++
	if _, exists := m.notes[note.ID]; !exists {
		return errors.New("note not found")
	}
	note.UpdatedAt = time.Now()
	m.notes[note.ID] = note
	return nil
}

func (m *mockNoteRepository) Delete(ctx context.Context, noteID uint64) error {
	m.deleteCount++
	if _, exists := m.notes[noteID]; !exists {
		return errors.New("note not found")
	}
	delete(m.notes, noteID)
	return nil
}

func (m *mockNoteRepository) CheckUserOwnership(ctx context.Context, noteID, userID uint64) (bool, error) {
	note, exists := m.notes[noteID]
	if !exists {
		return false, nil
	}
	return note.UserID == userID, nil
}

func TestNoteService_CreateNote(t *testing.T) {
	ctx := context.Background()
	repo := newMockNoteRepository()
	service := NewNoteService(repo)

	t.Run("successful creation", func(t *testing.T) {
		userID := uint64(1)
		title := "Test Note"
		content := "Test Content"

		note, err := service.CreateNote(ctx, userID, title, content, "")
		if err != nil {
			t.Fatalf("CreateNote failed: %v", err)
		}

		if note.Title != title {
			t.Errorf("Expected title %s, got %s", title, note.Title)
		}
		if note.Content != content {
			t.Errorf("Expected content %s, got %s", content, note.Content)
		}
		if note.UserID != userID {
			t.Errorf("Expected userID %d, got %d", userID, note.UserID)
		}
	})
}

func TestNoteService_GetNotes(t *testing.T) {
	ctx := context.Background()
	repo := newMockNoteRepository()
	service := NewNoteService(repo)

	userID := uint64(1)
	_, _ = service.CreateNote(ctx, userID, "Note 1", "Content 1", "")
	_, _ = service.CreateNote(ctx, userID, "Note 2", "Content 2", "")

	t.Run("get all notes for user", func(t *testing.T) {
		notes, err := service.GetNotes(ctx, userID)
		if err != nil {
			t.Fatalf("GetNotes failed: %v", err)
		}

		if len(notes) != 2 {
			t.Errorf("Expected 2 notes, got %d", len(notes))
		}
	})
}

func TestNoteService_GetNote(t *testing.T) {
	ctx := context.Background()
	repo := newMockNoteRepository()
	service := NewNoteService(repo)

	userID := uint64(1)
	note, _ := service.CreateNote(ctx, userID, "Test", "Content", "")

	t.Run("successful get", func(t *testing.T) {
		retrieved, err := service.GetNote(ctx, note.ID, userID)
		if err != nil {
			t.Fatalf("GetNote failed: %v", err)
		}

		if retrieved.ID != note.ID {
			t.Errorf("Expected ID %d, got %d", note.ID, retrieved.ID)
		}
	})

	t.Run("unauthorized access fails", func(t *testing.T) {
		otherUserID := uint64(2)
		_, err := service.GetNote(ctx, note.ID, otherUserID)
		if err == nil {
			t.Error("Expected error when accessing other user's note")
		}
	})
}

func TestNoteService_UpdateNote(t *testing.T) {
	ctx := context.Background()
	repo := newMockNoteRepository()
	service := NewNoteService(repo)

	userID := uint64(1)
	note, _ := service.CreateNote(ctx, userID, "Original", "Content", "")

	t.Run("successful update", func(t *testing.T) {
		updated, err := service.UpdateNote(ctx, note.ID, userID, "Updated", "New Content", "")
		if err != nil {
			t.Fatalf("UpdateNote failed: %v", err)
		}

		if updated.Title != "Updated" {
			t.Errorf("Expected title 'Updated', got %s", updated.Title)
		}
	})

	t.Run("update by non-owner fails", func(t *testing.T) {
		otherUserID := uint64(2)
		_, err := service.UpdateNote(ctx, note.ID, otherUserID, "Hacked", "Content", "")
		if err == nil {
			t.Error("Expected error when non-owner tries to update")
		}
	})
}

func TestNoteService_DeleteNote(t *testing.T) {
	ctx := context.Background()
	repo := newMockNoteRepository()
	service := NewNoteService(repo)

	userID := uint64(1)
	note, _ := service.CreateNote(ctx, userID, "Test", "Content", "")

	t.Run("successful delete", func(t *testing.T) {
		err := service.DeleteNote(ctx, note.ID, userID)
		if err != nil {
			t.Fatalf("DeleteNote failed: %v", err)
		}

		// Verify it's deleted
		_, err = service.GetNote(ctx, note.ID, userID)
		if err == nil {
			t.Error("Expected error when getting deleted note")
		}
	})

	t.Run("delete by non-owner fails", func(t *testing.T) {
		note2, _ := service.CreateNote(ctx, userID, "Test 2", "Content", "")
		otherUserID := uint64(2)

		err := service.DeleteNote(ctx, note2.ID, otherUserID)
		if err == nil {
			t.Error("Expected error when non-owner tries to delete")
		}
	})
}
