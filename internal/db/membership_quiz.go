package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/quiz"
	"github.com/jackc/pgx/v5"
)

type MembershipQuiz struct {
	Enabled       bool
	RoleID        *int64
	Tries         int
	CooldownHours int
	Questions     string
}

var (
	qMembershipQuiz = register("MembershipQuiz", `
SELECT enabled, role_id, tries, cooldown_hours, questions
FROM pwikit_membership_quiz WHERE site_id = $1`)

	qSaveMembershipQuiz = register("SaveMembershipQuiz", `
INSERT INTO pwikit_membership_quiz (site_id, enabled, role_id, tries, cooldown_hours, questions)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (site_id) DO UPDATE SET enabled = EXCLUDED.enabled, role_id = EXCLUDED.role_id,
	tries = EXCLUDED.tries, cooldown_hours = EXCLUDED.cooldown_hours, questions = EXCLUDED.questions`)

	qQuizAttempts = register("QuizAttempts", `
SELECT at, passed FROM pwikit_membership_quiz_attempt
WHERE site_id = $1 AND user_id = $2 ORDER BY at`)

	qAddQuizAttempt = register("AddQuizAttempt", `
INSERT INTO pwikit_membership_quiz_attempt (site_id, user_id, at, passed) VALUES ($1, $2, $3, $4)`)
)

// A site that never saved a quiz gets the defaults, switched off.
func (d *DB) MembershipQuiz(ctx context.Context, siteID int64) (MembershipQuiz, error) {
	q := MembershipQuiz{Tries: 3, CooldownHours: 12}
	err := d.pool.QueryRow(ctx, qMembershipQuiz, siteID).Scan(&q.Enabled, &q.RoleID, &q.Tries, &q.CooldownHours, &q.Questions)
	if errors.Is(err, pgx.ErrNoRows) {
		return q, nil
	}
	if err != nil {
		return q, fmt.Errorf("read the membership quiz of site %d: %w", siteID, err)
	}
	return q, nil
}

func (d *DB) SaveMembershipQuiz(ctx context.Context, siteID int64, q MembershipQuiz) error {
	if _, err := d.pool.Exec(ctx, qSaveMembershipQuiz, siteID, q.Enabled, q.RoleID, q.Tries, q.CooldownHours, q.Questions); err != nil {
		return fmt.Errorf("save the membership quiz of site %d: %w", siteID, err)
	}
	return nil
}

func (d *DB) QuizAttempts(ctx context.Context, siteID, userID int64) ([]quiz.Attempt, error) {
	rows, err := d.pool.Query(ctx, qQuizAttempts, siteID, userID)
	if err != nil {
		return nil, fmt.Errorf("read quiz attempts of user %d: %w", userID, err)
	}
	defer rows.Close()
	var out []quiz.Attempt
	for rows.Next() {
		var a quiz.Attempt
		if err := rows.Scan(&a.At, &a.Passed); err != nil {
			return nil, fmt.Errorf("scan a quiz attempt: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (d *DB) AddQuizAttempt(ctx context.Context, siteID, userID int64, at time.Time, passed bool) error {
	if _, err := d.pool.Exec(ctx, qAddQuizAttempt, siteID, userID, at, passed); err != nil {
		return fmt.Errorf("record a quiz attempt of user %d: %w", userID, err)
	}
	return nil
}
