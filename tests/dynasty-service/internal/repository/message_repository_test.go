package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/dynasty-service/internal/repository"
)

func TestMessageRepository_GetDynastyMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewMessageRepository(db)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("receiver_message").
			WillReturnRows(sqlmock.NewRows([]string{"message"}).AddRow("hello [sender-name]"))

		msg, err := repo.GetDynastyMessage(ctx, "receiver_message")
		require.NoError(t, err)
		assert.Equal(t, "hello [sender-name]", msg)
	})

	t.Run("ErrNoRows returns empty string", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("missing_type").
			WillReturnError(sql.ErrNoRows)

		msg, err := repo.GetDynastyMessage(ctx, "missing_type")
		require.NoError(t, err)
		assert.Equal(t, "", msg)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("broken").
			WillReturnError(errors.New("connection reset"))

		msg, err := repo.GetDynastyMessage(ctx, "broken")
		require.Error(t, err)
		assert.Empty(t, msg)
		assert.Contains(t, err.Error(), "failed to get dynasty message")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_FormatMessageWithPlaceholders(t *testing.T) {
	repo := repository.NewMessageRepository(nil)

	template := "From [sender-code]/[sender-name] to [reciever-code]/[reciever-name] as [relationship] on [created_at]"
	got := repo.FormatMessageWithPlaceholders(template, map[string]string{
		"[sender-code]":   "S1",
		"[reciever-code]": "R2",
		"[sender-name]":   "Alice",
		"[reciever-name]": "Bob",
		"[relationship]":  "برادر",
		"[created_at]":    "1403/01/01",
	})

	assert.Equal(t, "From S1/Alice to R2/Bob as برادر on 1403/01/01", got)
	assert.Equal(t, "", repo.FormatMessageWithPlaceholders("", map[string]string{"[sender-code]": "X"}))
	assert.Equal(t, "no placeholders", repo.FormatMessageWithPlaceholders("no placeholders", map[string]string{"[sender-code]": "X"}))
}

func TestMessageRepository_GetRelationshipTitle(t *testing.T) {
	repo := repository.NewMessageRepository(nil)

	cases := map[string]string{
		"brother":   "برادر",
		"sister":    "خواهر",
		"offspring": "فرزند",
		"father":    "پدر",
		"mother":    "مادر",
		"husband":   "شوهر",
		"wife":      "زن",
		"owner":     "مالک",
		"cousin":    "cousin", // unknown fallback
		"":          "",
	}

	for rel, want := range cases {
		assert.Equal(t, want, repo.GetRelationshipTitle(rel), "relationship=%q", rel)
	}
}

func TestMessageRepository_PrepareJoinRequestMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewMessageRepository(db)
	ctx := context.Background()

	t.Run("success with placeholder replacement", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("requester_confirmation_message").
			WillReturnRows(sqlmock.NewRows([]string{"message"}).
				AddRow("Sent to [reciever-code] ([reciever-name]) as [relationship] on [created_at] from [sender-code]/[sender-name]"))
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("reciever_message").
			WillReturnRows(sqlmock.NewRows([]string{"message"}).
				AddRow("From [sender-code] ([sender-name]) wants [relationship] on [created_at] to [reciever-name]"))

		senderMsg, receiverMsg, err := repo.PrepareJoinRequestMessages(
			ctx, "C1", "C2", "Alice", "Bob", "brother", "1403/05/01",
		)
		require.NoError(t, err)
		assert.Equal(t, "Sent to C2 (Bob) as برادر on 1403/05/01 from C1/Alice", senderMsg)
		assert.Equal(t, "From C1 (Alice) wants برادر on 1403/05/01 to Bob", receiverMsg)
	})

	t.Run("sender template DB error", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("requester_confirmation_message").
			WillReturnError(errors.New("db down"))

		_, _, err := repo.PrepareJoinRequestMessages(ctx, "C1", "C2", "A", "B", "sister", "d")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get sender template")
	})

	t.Run("receiver template DB error", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("requester_confirmation_message").
			WillReturnRows(sqlmock.NewRows([]string{"message"}).AddRow("ok"))
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("reciever_message").
			WillReturnError(errors.New("db down"))

		_, _, err := repo.PrepareJoinRequestMessages(ctx, "C1", "C2", "A", "B", "sister", "d")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get receiver template")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_PrepareAcceptMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewMessageRepository(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("requester_accept_message").
			WillReturnRows(sqlmock.NewRows([]string{"message"}).
				AddRow("Accepted: [sender-name] -> [reciever-name] ([relationship])"))
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("reciever_accept_message").
			WillReturnRows(sqlmock.NewRows([]string{"message"}).
				AddRow("You accepted [sender-code] as [relationship] on [created_at]"))

		reqMsg, recvMsg, err := repo.PrepareAcceptMessages(
			ctx, "R1", "R2", "Req", "Recv", "father", "1403/02/02",
		)
		require.NoError(t, err)
		assert.Equal(t, "Accepted: Req -> Recv (پدر)", reqMsg)
		assert.Equal(t, "You accepted R1 as پدر on 1403/02/02", recvMsg)
	})

	t.Run("requester template error", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("requester_accept_message").
			WillReturnError(errors.New("fail"))

		_, _, err := repo.PrepareAcceptMessages(ctx, "a", "b", "c", "d", "mother", "e")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get requester accept template")
	})

	t.Run("receiver template error", func(t *testing.T) {
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("requester_accept_message").
			WillReturnRows(sqlmock.NewRows([]string{"message"}).AddRow("ok"))
		mock.ExpectQuery("SELECT message FROM dynasty_messages").
			WithArgs("reciever_accept_message").
			WillReturnError(errors.New("fail"))

		_, _, err := repo.PrepareAcceptMessages(ctx, "a", "b", "c", "d", "mother", "e")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get receiver accept template")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_PrepareRejectMessages(t *testing.T) {
	repo := repository.NewMessageRepository(nil)

	reqMsg, recvMsg, err := repo.PrepareRejectMessages(context.Background(), "SENDER", "RECV")
	require.NoError(t, err)
	assert.Contains(t, reqMsg, "RECV")
	assert.Contains(t, reqMsg, "رد شد")
	assert.Contains(t, recvMsg, "SENDER")
	assert.Contains(t, recvMsg, "رد شد")
}
