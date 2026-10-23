package store

import (
	"context"
	"database/sql"
	"time"
)

// maxComments is how many of a project's latest comments Comments returns.
const maxComments = 200

// Comments returns a project's latest comments, oldest first.
func (s *Store) Comments(ctx context.Context, projectID int64) ([]Comment, error) {
	if err := s.projectExists(ctx, projectID); err != nil {
		return nil, err
	}
	return queryList(ctx, s.db, `
		SELECT id, name, body, created_at FROM (
			SELECT * FROM project_comments WHERE project_id = ? ORDER BY id DESC LIMIT ?
		) ORDER BY id`,
		func(rows *sql.Rows, c *Comment) error {
			var created int64
			if err := rows.Scan(&c.ID, &c.Name, &c.Body, &created); err != nil {
				return err
			}
			c.CreatedAt = time.Unix(created, 0).UTC()
			return nil
		}, projectID, maxComments)
}

// InsertComment adds a comment to a project, setting its ID and time.
func (s *Store) InsertComment(ctx context.Context, projectID int64, c *Comment) (err error) {
	if err := s.projectExists(ctx, projectID); err != nil {
		return err
	}
	c.CreatedAt = time.Now().UTC().Truncate(time.Second)
	c.ID, err = insertedID(s.db.ExecContext(ctx,
		`INSERT INTO project_comments (project_id, name, body, created_at) VALUES (?, ?, ?, ?)`,
		projectID, c.Name, c.Body, c.CreatedAt.Unix()))
	return err
}

func (s *Store) DeleteComment(ctx context.Context, id int64) error {
	return updated(s.db.ExecContext(ctx, `DELETE FROM project_comments WHERE id = ?`, id))
}
