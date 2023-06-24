package store

import (
	"context"
	"database/sql"
	"strings"
)

type ActivityType string

const (
	// ActivityShortcutView is the activity type of shortcut create.
	ActivityShortcutCreate ActivityType = "shortcut.create"
	// ActivityShortcutView is the activity type of shortcut view.
	ActivityShortcutView ActivityType = "shortcut.view"
)

func (t ActivityType) String() string {
	switch t {
	case ActivityShortcutCreate:
		return "shortcut.create"
	case ActivityShortcutView:
		return "shortcut.view"
	}
	return ""
}

type ActivityLevel string

const (
	// ActivityInfo is the activity level of info.
	ActivityInfo ActivityLevel = "INFO"
	// ActivityWarn is the activity level of warn.
	ActivityWarn ActivityLevel = "WARN"
	// ActivityError is the activity level of error.
	ActivityError ActivityLevel = "ERROR"
)

func (l ActivityLevel) String() string {
	switch l {
	case ActivityInfo:
		return "INFO"
	case ActivityWarn:
		return "WARN"
	case ActivityError:
		return "ERROR"
	}
	return ""
}

type Activity struct {
	ID        int
	CreatorID int
	CreatedTs int64
	Type      ActivityType
	Level     ActivityLevel
	Payload   string
}

type FindActivity struct {
	Type  ActivityType
	Level ActivityLevel
}

func (s *Store) CreateActivity(ctx context.Context, create *Activity) (*Activity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO activity (
			creator_id,
			type,
			level,
			payload
		)
		VALUES (?, ?, ?, ?)
		RETURNING id, created_ts
	`
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.Type.String(),
		create.Level.String(),
		create.Payload,
	).Scan(
		&create.ID,
		&create.CreatedTs,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	activity := create
	return activity, nil
}

func (s *Store) ListActivities(ctx context.Context, find *FindActivity) ([]*Activity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := listActivities(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (s *Store) GetActivity(ctx context.Context, find *FindActivity) (*Activity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := listActivities(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	}

	activity := list[0]
	return activity, nil
}

func listActivities(ctx context.Context, tx *sql.Tx, find *FindActivity) ([]*Activity, error) {
	where, args := []string{"1 = 1"}, []any{}
	if find.Type != "" {
		where, args = append(where, "type = ?"), append(args, find.Type.String())
	}
	if find.Level != "" {
		where, args = append(where, "level = ?"), append(args, find.Level.String())
	}

	query := `
		SELECT
			id,
			creator_id,
			created_ts,
			type,
			level,
			payload
		FROM activity
		WHERE ` + strings.Join(where, " AND ")
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	list := []*Activity{}
	for rows.Next() {
		activity := &Activity{}
		if err := rows.Scan(
			&activity.ID,
			&activity.CreatorID,
			&activity.CreatedTs,
			&activity.Type,
			&activity.Level,
			&activity.Payload,
		); err != nil {
			return nil, err
		}

		list = append(list, activity)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}