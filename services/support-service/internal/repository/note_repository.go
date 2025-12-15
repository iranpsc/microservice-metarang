package repository

import (
	"context"
	"database/sql"
	"fmt"
	"metargb/support-service/internal/models"
)

type NoteRepository interface {
	Create(ctx context.Context, note *models.Note) (*models.Note, error)
	GetByID(ctx context.Context, noteID uint64) (*models.Note, error)
	GetByUserID(ctx context.Context, userID uint64) ([]*models.Note, error)
	Update(ctx context.Context, note *models.Note) error
	Delete(ctx context.Context, noteID uint64) error
	CheckUserOwnership(ctx context.Context, noteID, userID uint64) (bool, error)
}

type noteRepository struct {
	db *sql.DB
}

func NewNoteRepository(db *sql.DB) NoteRepository {
	return &noteRepository{db: db}
}

func (r *noteRepository) Create(ctx context.Context, note *models.Note) (*models.Note, error) {
	query := `
		INSERT INTO notes (title, content, attachment, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
	`

	result, err := r.db.ExecContext(ctx, query,
		note.Title,
		note.Content,
		note.Attachment,
		note.UserID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	note.ID = uint64(id)
	return note, nil
}

func (r *noteRepository) GetByID(ctx context.Context, noteID uint64) (*models.Note, error) {
	query := `
		SELECT id, title, content, attachment, user_id, created_at, updated_at
		FROM notes
		WHERE id = ?
	`

	var note models.Note
	err := r.db.QueryRowContext(ctx, query, noteID).Scan(
		&note.ID, &note.Title, &note.Content, &note.Attachment,
		&note.UserID, &note.CreatedAt, &note.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	return &note, nil
}

func (r *noteRepository) GetByUserID(ctx context.Context, userID uint64) ([]*models.Note, error) {
	query := `
		SELECT id, title, content, attachment, user_id, created_at, updated_at
		FROM notes
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		var note models.Note
		err := rows.Scan(
			&note.ID, &note.Title, &note.Content, &note.Attachment,
			&note.UserID, &note.CreatedAt, &note.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, &note)
	}

	return notes, nil
}

func (r *noteRepository) Update(ctx context.Context, note *models.Note) error {
	query := `
		UPDATE notes 
		SET title = ?, content = ?, attachment = ?, updated_at = NOW()
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		note.Title,
		note.Content,
		note.Attachment,
		note.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	return nil
}

func (r *noteRepository) Delete(ctx context.Context, noteID uint64) error {
	query := `DELETE FROM notes WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, noteID)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	return nil
}

func (r *noteRepository) CheckUserOwnership(ctx context.Context, noteID, userID uint64) (bool, error) {
	query := `SELECT COUNT(*) FROM notes WHERE id = ? AND user_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, noteID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check ownership: %w", err)
	}

	return count > 0, nil
}
