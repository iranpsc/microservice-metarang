package repository

import (
	"context"
	"database/sql"
	"fmt"

	pb "metargb/shared/pb/levels"
)

// ChallengeRepository handles questions and answers
// Implements Laravel's Challenge\Question, Challenge\Answer, Challenge\UserQuestionAnswer models
type ChallengeRepository struct {
	db *sql.DB
}

func NewChallengeRepository(db *sql.DB) *ChallengeRepository {
	return &ChallengeRepository{db: db}
}

// GetRandomUnansweredQuestion retrieves a random question the user hasn't answered
// Implements Laravel: while loop in ChallengeController@selectQuestion
func (r *ChallengeRepository) GetRandomUnansweredQuestion(ctx context.Context, userID uint64) (*pb.Question, error) {
	// Try to find a random unanswered question
	query := `
		SELECT q.id, q.text, q.prize, q.views, q.participants
		FROM questions q
		WHERE q.id NOT IN (
			SELECT question_id FROM user_question_answers WHERE user_id = ?
		)
		ORDER BY RAND()
		LIMIT 1
	`
	
	var question pb.Question
	
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&question.Id,
		&question.Text,
		&question.Prize,
		&question.Views,
		&question.Participants,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No unanswered questions
		}
		return nil, err
	}
	
	// Load answers for the question
	answers, err := r.GetAnswersForQuestion(ctx, question.Id)
	if err != nil {
		return nil, err
	}
	
	question.Answers = answers
	
	return &question, nil
}

// GetAnswersForQuestion retrieves all answers for a question (without is_correct field for security)
// Implements Laravel: $question->load('answers')
func (r *ChallengeRepository) GetAnswersForQuestion(ctx context.Context, questionID uint64) ([]*pb.Answer, error) {
	query := `
		SELECT id, question_id, text
		FROM answers
		WHERE question_id = ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var answers []*pb.Answer
	for rows.Next() {
		var answer pb.Answer
		if err := rows.Scan(&answer.Id, &answer.QuestionId, &answer.Text); err != nil {
			return nil, err
		}
		answers = append(answers, &answer)
	}
	
	return answers, nil
}

// IncrementViews increments question views
// Implements Laravel: $question->increment('views')
func (r *ChallengeRepository) IncrementViews(ctx context.Context, questionID uint64) error {
	query := "UPDATE questions SET views = views + 1 WHERE id = ?"
	_, err := r.db.ExecContext(ctx, query, questionID)
	return err
}

// ValidateAnswer checks if answer belongs to question
// Implements Laravel: if ($answer->question->isNot($question))
func (r *ChallengeRepository) ValidateAnswer(ctx context.Context, questionID, answerID uint64) (bool, error) {
	query := "SELECT COUNT(*) FROM answers WHERE id = ? AND question_id = ?"
	var count int
	err := r.db.QueryRowContext(ctx, query, answerID, questionID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// RecordUserAnswer records user's answer
// Implements Laravel: UserQuestionAnswer::create([...])
func (r *ChallengeRepository) RecordUserAnswer(ctx context.Context, userID, questionID, answerID uint64) error {
	query := `
		INSERT INTO user_question_answers (user_id, question_id, answer_id, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
	`
	_, err := r.db.ExecContext(ctx, query, userID, questionID, answerID)
	return err
}

// IncrementParticipants increments question participants
// Implements Laravel: $question->increment('participants')
func (r *ChallengeRepository) IncrementParticipants(ctx context.Context, questionID uint64) error {
	query := "UPDATE questions SET participants = participants + 1 WHERE id = ?"
	_, err := r.db.ExecContext(ctx, query, questionID)
	return err
}

// CheckAnswer checks if answer is correct and returns prize
// Implements Laravel: $answer->isCorrect() and $question->prize
func (r *ChallengeRepository) CheckAnswer(ctx context.Context, answerID, questionID uint64) (bool, string, error) {
	query := `
		SELECT a.is_correct, q.prize
		FROM answers a
		JOIN questions q ON a.question_id = q.id
		WHERE a.id = ? AND a.question_id = ?
	`
	
	var isCorrect bool
	var prize string
	
	err := r.db.QueryRowContext(ctx, query, answerID, questionID).Scan(&isCorrect, &prize)
	if err != nil {
		return false, "0", err
	}
	
	return isCorrect, prize, nil
}

// GetQuestionByID retrieves question with its answers
// Implements Laravel: Question::findOrFail($id)->load('answers')
func (r *ChallengeRepository) GetQuestionByID(ctx context.Context, questionID uint64) (*pb.Question, error) {
	query := `
		SELECT id, text, prize, views, participants
		FROM questions
		WHERE id = ?
	`
	
	var question pb.Question
	
	err := r.db.QueryRowContext(ctx, query, questionID).Scan(
		&question.Id,
		&question.Text,
		&question.Prize,
		&question.Views,
		&question.Participants,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Load answers
	answers, err := r.GetAnswersForQuestion(ctx, question.Id)
	if err != nil {
		return nil, err
	}
	
	question.Answers = answers
	
	return &question, nil
}

// GetChallengeIntervals retrieves challenge timing intervals from system variables
// Implements Laravel: SystemVariable::getByKey('challenge_display_ad_interval') ?? 15
func (r *ChallengeRepository) GetChallengeIntervals(ctx context.Context) (int32, int32, int32, error) {
	query := `
		SELECT 
			COALESCE(MAX(CASE WHEN key_name = 'challenge_display_ad_interval' THEN value END), '15') as ad_interval,
			COALESCE(MAX(CASE WHEN key_name = 'challenge_display_question_interval' THEN value END), '15') as question_interval,
			COALESCE(MAX(CASE WHEN key_name = 'challenge_display_answer_interval' THEN value END), '15') as answer_interval
		FROM system_variables
		WHERE key_name IN ('challenge_display_ad_interval', 'challenge_display_question_interval', 'challenge_display_answer_interval')
	`
	
	var adInterval, questionInterval, answerInterval string
	
	err := r.db.QueryRowContext(ctx, query).Scan(&adInterval, &questionInterval, &answerInterval)
	if err != nil {
		return 15, 15, 15, nil // Return defaults on error
	}
	
	var ad, question, answer int32
	_, _ = fmt.Sscanf(adInterval, "%d", &ad)
	_, _ = fmt.Sscanf(questionInterval, "%d", &question)
	_, _ = fmt.Sscanf(answerInterval, "%d", &answer)
	
	if ad == 0 {
		ad = 15
	}
	if question == 0 {
		question = 15
	}
	if answer == 0 {
		answer = 15
	}
	
	return ad, question, answer, nil
}

// GetUserAnswerCounts retrieves user's correct and wrong answer counts
// Implements Laravel: ChallengeController@getCorrectAnswers and @getWrongAnswers
func (r *ChallengeRepository) GetUserAnswerCounts(ctx context.Context, userID uint64) (int32, int32, error) {
	query := `
		SELECT 
			SUM(CASE WHEN a.is_correct = 1 THEN 1 ELSE 0 END) as correct_count,
			SUM(CASE WHEN a.is_correct = 0 THEN 1 ELSE 0 END) as wrong_count
		FROM user_question_answers uqa
		JOIN answers a ON uqa.answer_id = a.id
		WHERE uqa.user_id = ?
	`
	
	var correct, wrong sql.NullInt32
	
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&correct, &wrong)
	if err != nil {
		return 0, 0, err
	}
	
	correctCount := int32(0)
	wrongCount := int32(0)
	
	if correct.Valid {
		correctCount = correct.Int32
	}
	if wrong.Valid {
		wrongCount = wrong.Int32
	}
	
	return correctCount, wrongCount, nil
}

// GetTotalParticipants retrieves total unique participants
// Implements Laravel: UserQuestionAnswer::distinct()->count('user_id')
func (r *ChallengeRepository) GetTotalParticipants(ctx context.Context) (int32, error) {
	query := "SELECT COUNT(DISTINCT user_id) FROM user_question_answers"
	var count int32
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// HasUserAnsweredQuestion checks if user has already answered a question
// Used for authorization (Laravel Policy)
func (r *ChallengeRepository) HasUserAnsweredQuestion(ctx context.Context, userID, questionID uint64) (bool, error) {
	query := "SELECT COUNT(*) FROM user_question_answers WHERE user_id = ? AND question_id = ?"
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, questionID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

