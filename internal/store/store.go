// Package store keeps the portfolio's content and sign-in sessions in SQLite.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

//go:embed schema.sql
var schema string

// Content for a fresh database, converted from the pages' old data.js.
//
//go:embed seed.json
var seedJSON []byte

type Store struct {
	db *sql.DB
}

// execer is satisfied by both *sql.DB and *sql.Tx, so inserts can run inside the seed transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Open opens the database at path, creating the tables and seeding it if it's new.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	// One connection: plenty for a portfolio, and SQLite never reports "database is locked".
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	if err := s.seed(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("seed: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// seed fills an empty database from seed.json.
func (s *Store) seed(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profile`).Scan(&count); err != nil || count > 0 {
		return err
	}

	var p Portfolio
	if err := json.Unmarshal(seedJSON, &p); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := saveProfile(ctx, tx, p.Profile, p.Education); err != nil {
		return err
	}
	for i := range p.Skills {
		if err := insertSkill(ctx, tx, &p.Skills[i]); err != nil {
			return err
		}
	}
	for i := range p.Projects {
		if err := insertProject(ctx, tx, &p.Projects[i]); err != nil {
			return err
		}
	}
	for i := range p.Experience {
		if err := insertExperience(ctx, tx, &p.Experience[i]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---- Reads ----

func (s *Store) Portfolio(ctx context.Context) (Portfolio, error) {
	var p Portfolio
	err := s.db.QueryRowContext(ctx, `
		SELECT name, greeting, handle, initials, photo, tagline, status, location, phone, email,
			github, linkedin, edu_degree, edu_school, edu_year
		FROM profile WHERE id = 1`,
	).Scan(&p.Profile.Name, &p.Profile.Greeting, &p.Profile.Handle, &p.Profile.Initials, &p.Profile.Photo,
		&p.Profile.Tagline, &p.Profile.Status, &p.Profile.Location, &p.Profile.Phone, &p.Profile.Email,
		&p.Profile.GitHub, &p.Profile.LinkedIn, &p.Education.Degree, &p.Education.School, &p.Education.Year)
	if err != nil {
		return p, err
	}

	if p.Skills, err = queryList(ctx, s.db, `SELECT id, label, items FROM skills ORDER BY position, id`,
		func(rows *sql.Rows, skill *Skill) error {
			var items string
			if err := rows.Scan(&skill.ID, &skill.Label, &items); err != nil {
				return err
			}
			return json.Unmarshal([]byte(items), &skill.Items)
		}); err != nil {
		return p, err
	}

	if p.Projects, err = queryList(ctx, s.db,
		`SELECT id, name, type, language, description, stack, image, url FROM projects ORDER BY position, id`,
		func(rows *sql.Rows, project *Project) error {
			var stack string
			if err := rows.Scan(&project.ID, &project.Name, &project.Type, &project.Language,
				&project.Description, &stack, &project.Image, &project.URL); err != nil {
				return err
			}
			return json.Unmarshal([]byte(stack), &project.Stack)
		}); err != nil {
		return p, err
	}

	p.Experience, err = queryList(ctx, s.db,
		`SELECT id, org, meta, roles, highlights FROM experience ORDER BY position, id`,
		func(rows *sql.Rows, exp *Experience) error {
			var roles, highlights string
			if err := rows.Scan(&exp.ID, &exp.Org, &exp.Meta, &roles, &highlights); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(roles), &exp.Roles); err != nil {
				return err
			}
			return json.Unmarshal([]byte(highlights), &exp.Highlights)
		})
	return p, err
}

// queryList runs query and scans each row with scan. It returns an empty
// (not nil) slice for no rows, so the JSON is [] rather than null.
func queryList[T any](ctx context.Context, db *sql.DB, query string, scan func(*sql.Rows, *T) error) ([]T, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []T{}
	for rows.Next() {
		var item T
		if err := scan(rows, &item); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ---- Writes ----

func (s *Store) SaveProfile(ctx context.Context, p Profile, e Education) error {
	return saveProfile(ctx, s.db, p, e)
}

func saveProfile(ctx context.Context, db execer, p Profile, e Education) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO profile (id, name, greeting, handle, initials, photo, tagline, status, location, phone,
			email, github, linkedin, edu_degree, edu_school, edu_year)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			name = excluded.name, greeting = excluded.greeting, handle = excluded.handle,
			initials = excluded.initials, photo = excluded.photo, tagline = excluded.tagline,
			status = excluded.status, location = excluded.location, phone = excluded.phone,
			email = excluded.email, github = excluded.github, linkedin = excluded.linkedin,
			edu_degree = excluded.edu_degree, edu_school = excluded.edu_school, edu_year = excluded.edu_year`,
		p.Name, p.Greeting, p.Handle, p.Initials, p.Photo, p.Tagline, p.Status, p.Location, p.Phone,
		p.Email, p.GitHub, p.LinkedIn, e.Degree, e.School, e.Year)
	return err
}

func (s *Store) InsertSkill(ctx context.Context, skill *Skill) error {
	return insertSkill(ctx, s.db, skill)
}

func insertSkill(ctx context.Context, db execer, skill *Skill) (err error) {
	skill.ID, err = insertedID(db.ExecContext(ctx, `
		INSERT INTO skills (position, label, items)
		VALUES ((SELECT COALESCE(MAX(position), 0) + 1 FROM skills), ?, ?)`,
		skill.Label, toJSON(skill.Items)))
	return err
}

func (s *Store) UpdateSkill(ctx context.Context, id int64, skill *Skill) error {
	skill.ID = id
	return updated(s.db.ExecContext(ctx, `UPDATE skills SET label = ?, items = ? WHERE id = ?`,
		skill.Label, toJSON(skill.Items), id))
}

func (s *Store) DeleteSkill(ctx context.Context, id int64) error {
	return updated(s.db.ExecContext(ctx, `DELETE FROM skills WHERE id = ?`, id))
}

func (s *Store) InsertProject(ctx context.Context, p *Project) error {
	return insertProject(ctx, s.db, p)
}

func insertProject(ctx context.Context, db execer, p *Project) (err error) {
	p.ID, err = insertedID(db.ExecContext(ctx, `
		INSERT INTO projects (position, name, type, language, description, stack, image, url)
		VALUES ((SELECT COALESCE(MAX(position), 0) + 1 FROM projects), ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Type, p.Language, p.Description, toJSON(p.Stack), p.Image, p.URL))
	return err
}

func (s *Store) UpdateProject(ctx context.Context, id int64, p *Project) error {
	p.ID = id
	return updated(s.db.ExecContext(ctx, `
		UPDATE projects SET name = ?, type = ?, language = ?, description = ?, stack = ?, image = ?, url = ?
		WHERE id = ?`,
		p.Name, p.Type, p.Language, p.Description, toJSON(p.Stack), p.Image, p.URL, id))
}

func (s *Store) DeleteProject(ctx context.Context, id int64) error {
	return updated(s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id))
}

func (s *Store) InsertExperience(ctx context.Context, e *Experience) error {
	return insertExperience(ctx, s.db, e)
}

func insertExperience(ctx context.Context, db execer, e *Experience) (err error) {
	e.ID, err = insertedID(db.ExecContext(ctx, `
		INSERT INTO experience (position, org, meta, roles, highlights)
		VALUES ((SELECT COALESCE(MAX(position), 0) + 1 FROM experience), ?, ?, ?, ?)`,
		e.Org, e.Meta, toJSON(e.Roles), toJSON(e.Highlights)))
	return err
}

func (s *Store) UpdateExperience(ctx context.Context, id int64, e *Experience) error {
	e.ID = id
	return updated(s.db.ExecContext(ctx, `UPDATE experience SET org = ?, meta = ?, roles = ?, highlights = ? WHERE id = ?`,
		e.Org, e.Meta, toJSON(e.Roles), toJSON(e.Highlights), id))
}

func (s *Store) DeleteExperience(ctx context.Context, id int64) error {
	return updated(s.db.ExecContext(ctx, `DELETE FROM experience WHERE id = ?`, id))
}

// insertedID returns the ID of the row an INSERT added.
func insertedID(res sql.Result, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// updated turns "no rows affected" into ErrNotFound.
func updated(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

// toJSON encodes a list for a JSON text column; nil becomes [].
func toJSON[T any](list []T) string {
	if list == nil {
		return "[]"
	}
	b, _ := json.Marshal(list)
	return string(b)
}
