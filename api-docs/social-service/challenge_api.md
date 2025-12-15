# Challenge API (v1)

The Challenge API exposes the endpoints required to run the in-app quiz/challenge experience. It lets authenticated users discover timing configuration, fetch an unanswered question, and submit their answer while the platform tracks participation statistics and rewards winners.

- **Base path:** `/api/challenge`
- **Middleware:** `auth:sanctum`, `verified`, `activity`
- **Namespace:** `App\Http\Controllers\Api\V1\ChallengeController`

All endpoints require a valid Sanctum token tied to a verified and active account.

## Shared Response Models

### Question Resource

Questions are serialized through `App\Http\Resources\QuestionResource` and include:

```0:8:app/Http/Resources/QuestionResource.php
return [
    'id' => $this->id,
    'title' => $this->title,
    'image' => $this->image,                   // Resolved via config('rgb.ftp-endpoint')
    'prize' => $this->prize,
    'participants' => $this->participants,
    'views' => $this->views,
    'creator_code' => $this->creator_code,
    'answers' => AnswerResource::collection($this->answers),
];
```

- Answer entries expose `id`, `title`, `image`.
- When the response originates from the answer submission route (`challenge.answer`), answers additionally include `is_correct` and `vote_percentage`.

### Timing Metrics

`GET /challenge/timings` returns aggregate statistics plus display interval configuration. Interval values are pulled from the `system_variables` table using the keys:

- `challenge_display_ad_interval`
- `challenge_display_question_interval`
- `challenge_display_answer_interval`

Each key falls back to `15` seconds when the system variable is missing.

## Endpoints

### GET `/challenge/timings`

Retrieves the configured display intervals and live participation statistics.

- **Controller method:** `ChallengeController@getTimings`
- **Response:** `200 OK`

#### Response body

```json
{
  "data": {
    "display_ad_interval": 15,
    "display_question_interval": 30,
    "display_answer_interval": 20,
    "participants": 1280,
    "correct_answers": 512,
    "wrong_answers": 768
  }
}
```

`participants` counts distinct users who have submitted at least one answer. `correct_answers` and `wrong_answers` are scoped to the currently authenticated user.

### POST `/challenge/question`

Returns a random question the current user has not yet answered correctly. When the user has previously selected the wrong answer, the same question may be returned so they can attempt it again.

- **Controller method:** `ChallengeController@getQuestion`
- **Response:** `200 OK`

#### Behaviour

1. Picks a random question (`Question::inRandomOrder()`).
2. Skips questions the user has already answered correctly (via `QuestionPolicy`).
3. Increments the question's `views` counter before returning it.
4. Returns `null` when no suitable question exists.

#### Sample response

```json
{
  "id": 42,
  "title": "What is the capital of France?",
  "image": "https://cdn.example.com/public/challenge/question-image.png",
  "prize": 25,
  "participants": 734,
  "views": 1980,
  "creator_code": "USR-9876",
  "answers": [
    {
      "id": 4201,
      "title": "Paris",
      "image": "https://cdn.example.com/public/challenge/answer-4201.png"
    },
    {
      "id": 4202,
      "title": "Lyon",
      "image": "https://cdn.example.com/public/challenge/answer-4202.png"
    }
  ]
}
```

### POST `/challenge/answer`

Validates and records a user's answer, returning the full question resource enriched with answer meta-data.

- **Controller method:** `ChallengeController@answerResult`
- **Response:** `200 OK` with `QuestionResource`

#### Request body

| Field        | Type    | Rules                                    | Description                         |
|--------------|---------|------------------------------------------|-------------------------------------|
| `question_id`| integer | `required`, `integer`, `exists:questions,id` | ID of the question being answered.  |
| `answer_id`  | integer | `required`, `integer`, `exists:answers,id`   | ID of the selected answer option.   |

#### Validation & Authorization

- **Validation:** Laravel validator enforces presence and referential integrity for both IDs.
- **Additional guard:** Throws `ValidationException` if the provided answer does not belong to the given question.
- **Policy:** Calls `$this->authorize('answer', $question)`.

```0:9:app/Policies/QuestionPolicy.php
public function answer(User $user, Question $question): bool
{
    return UserQuestionAnswer::whereUserId($user->id)
        ->whereQuestionId($question->id)
        ->whereAnswerId($question->correctAnswer->id)
        ->doesntExist();
}
```

Users are blocked (`403 Forbidden`) if they have already answered the question correctly.

#### Side effects

- Saves a `UserQuestionAnswer` record tied to the authenticated user.
- Increments the question's `participants` counter.
- When the answer is correct (`Answer::isCorrect()`), increments the user's `psc` wallet balance by the question's `prize`.

#### Successful response preview

```json
{
  "id": 42,
  "title": "What is the capital of France?",
  "image": "https://cdn.example.com/public/challenge/question-image.png",
  "prize": 25,
  "participants": 735,
  "views": 1980,
  "creator_code": "USR-9876",
  "answers": [
    {
      "id": 4201,
      "title": "Paris",
      "image": "https://cdn.example.com/public/challenge/answer-4201.png",
      "is_correct": true,
      "vote_percentage": 76
    },
    {
      "id": 4202,
      "title": "Lyon",
      "image": "https://cdn.example.com/public/challenge/answer-4202.png",
      "is_correct": false,
      "vote_percentage": 24
    }
  ]
}
```

`vote_percentage` expresses the share of participants who have selected each answer so far (rounded down).

#### Error scenarios

- `422 Unprocessable Entity` when validation fails (missing IDs, invalid references, mismatched question/answer).
- `403 Forbidden` when the `QuestionPolicy@answer` check fails (user already has a correct response on record).

## Data Flow Summary

1. **Timings** provide configuration and aggregates via `SystemVariable` lookups and `UserQuestionAnswer` counts.
2. **Question selection** skips questions the user has cleared correctly while allowing retries on incorrect attempts.
3. **Answer submission** verifies ownership, enforces policy rules, tracks participation, and rewards winners by crediting their `psc` wallet.

## Testing Checklist

- Ensure requests include a valid Sanctum token and hit `/api/challenge/*`.
- Verify the wallet balance increases only when the answer payload matches the question's correct answer.
- Confirm previously completed questions return `403` on repeat submissions and are no longer served by `/challenge/question`.


