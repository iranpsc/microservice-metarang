package models

import "time"

// Question represents a challenge question
// Maps to Laravel: App\Models\Challenge\Question
type Question struct {
	ID           uint64    `json:"id" db:"id"`
	Text         string    `json:"text" db:"text"`
	Prize        string    `json:"prize" db:"prize"` // PSC amount as string
	Views        int32     `json:"views" db:"views"`
	Participants int32     `json:"participants" db:"participants"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Answer represents a possible answer to a question
// Maps to Laravel: App\Models\Challenge\Answer
type Answer struct {
	ID         uint64    `json:"id" db:"id"`
	QuestionID uint64    `json:"question_id" db:"question_id"`
	Text       string    `json:"text" db:"text"`
	IsCorrect  bool      `json:"is_correct" db:"is_correct"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// UserQuestionAnswer represents user's answers to questions
// Maps to Laravel: App\Models\Challenge\UserQuestionAnswer
type UserQuestionAnswer struct {
	ID         uint64    `json:"id" db:"id"`
	UserID     uint64    `json:"user_id" db:"user_id"`
	QuestionID uint64    `json:"question_id" db:"question_id"`
	AnswerID   uint64    `json:"answer_id" db:"answer_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

