-- Small lists (stack, items, roles, highlights) are stored as JSON text.
-- Lists are shown in `position` order; new rows go last.

CREATE TABLE IF NOT EXISTS profile (
	id         INTEGER PRIMARY KEY CHECK (id = 1),
	name       TEXT NOT NULL DEFAULT '',
	greeting   TEXT NOT NULL DEFAULT '',
	handle     TEXT NOT NULL DEFAULT '',
	initials   TEXT NOT NULL DEFAULT '',
	photo      TEXT NOT NULL DEFAULT '',
	tagline    TEXT NOT NULL DEFAULT '',
	status     TEXT NOT NULL DEFAULT '',
	location   TEXT NOT NULL DEFAULT '',
	phone      TEXT NOT NULL DEFAULT '',
	email      TEXT NOT NULL DEFAULT '',
	github     TEXT NOT NULL DEFAULT '',
	linkedin   TEXT NOT NULL DEFAULT '',
	edu_degree TEXT NOT NULL DEFAULT '',
	edu_school TEXT NOT NULL DEFAULT '',
	edu_year   TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS projects (
	id          INTEGER PRIMARY KEY,
	position    INTEGER NOT NULL,
	name        TEXT NOT NULL,
	type        TEXT NOT NULL DEFAULT '',
	language    TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	details     TEXT NOT NULL DEFAULT '',
	stack       TEXT NOT NULL DEFAULT '[]',
	image       TEXT NOT NULL DEFAULT '',
	gallery     TEXT NOT NULL DEFAULT '[]',
	url         TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS skills (
	id       INTEGER PRIMARY KEY,
	position INTEGER NOT NULL,
	label    TEXT NOT NULL,
	items    TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS experience (
	id         INTEGER PRIMARY KEY,
	position   INTEGER NOT NULL,
	org        TEXT NOT NULL,
	meta       TEXT NOT NULL DEFAULT '',
	roles      TEXT NOT NULL DEFAULT '[]',
	highlights TEXT NOT NULL DEFAULT '[]'
);

-- One row per visitor per starred project; `visitor` is a hash of the ID the visitor's browser made up.
CREATE TABLE IF NOT EXISTS project_stars (
	project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
	visitor    TEXT NOT NULL,
	PRIMARY KEY (project_id, visitor)
);

-- Visitors' comments on projects, shown oldest first.
CREATE TABLE IF NOT EXISTS project_comments (
	id         INTEGER PRIMARY KEY,
	project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
	name       TEXT NOT NULL,
	body       TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS project_comments_by_project ON project_comments (project_id, id);

CREATE TABLE IF NOT EXISTS sessions (
	token_hash TEXT PRIMARY KEY,
	expires_at INTEGER NOT NULL
);
