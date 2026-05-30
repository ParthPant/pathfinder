package memory

const CREATE_METADATA_TABLE = `CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,

    kind TEXT NOT NULL,
	name TEXT NOT NULL,
    content TEXT NOT NULL,
    tags TEXT,                       -- JSON array or comma-separated

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted INTEGER DEFAULT 0,
    version INTEGER DEFAULT 1,
    path TEXT                        -- markdown/json backing file
);`

const CREATE_VIRTUAL_TABLE = `CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
	name,
    content,
    tags,

    content='memories',
    content_rowid='rowid',

    tokenize='porter unicode61'
);`

const INSERT_TRIGGER = `CREATE TRIGGER IF NOT EXISTS memories_ai
AFTER INSERT ON memories
BEGIN
    INSERT INTO memories_fts(
        rowid,
        name,
        content,
        tags
    )
    VALUES (
        new.rowid,
        new.name,
        new.content,
        new.tags
    );
END;`

const UPDATE_TRIGGER = `CREATE TRIGGER IF NOT EXISTS memories_au
AFTER UPDATE ON memories
BEGIN
    INSERT INTO memories_fts(
        memories_fts,
        name,
        content,
        tags
    )
    VALUES (
        'delete',
        old.rowid,
		old.name,
        old.content,
        old.tags
    );

    INSERT INTO memories_fts(
        rowid,
		name,
        content,
        tags
    )
    VALUES (
        new.rowid,
		new.name,
        new.content,
        new.tags
    );
END;`

const DELETE_TRIGGER = `CREATE TRIGGER IF NOT EXISTS memories_ad
AFTER DELETE ON memories
BEGIN
    INSERT INTO memories_fts(
        memories_fts,
        rowid,
        name,
        content,
        tags
    )
    VALUES (
        'delete',
        old.rowid,
		old.name,
        old.content,
        old.tags
    );
END;`

const INSERT_QUERY = `INSERT into memories(
	id,
	kind,
	name,
	content,
	tags,
	updated_at,
	created_at,
	deleted,
	version,
	path
) VALUES (
	?1, -- id
	?2, -- kind
	?3, -- name
	?4, -- content
	?5, -- tags
	?6, -- updated_at
	?7, -- creaed_at
	?8, -- deleted
	?9, -- version
	?10 -- path
);`

const SEARCH_QUERY = `SELECT
    m.id,
    m.kind,
	m.name,
    m.content,
    m.tags,
    bm25(memories_fts) AS rank
FROM memories_fts
JOIN memories m
    ON memories_fts.rowid = m.rowid
WHERE memories_fts MATCH ?
AND m.deleted = 0
ORDER BY rank
LIMIT 10;`

const FETCH_ALL_LATEST_QUERY = `SELECT m.id, m.kind, m.name, m.content, m.tags, m.created_at, m.updated_at, m.deleted, m.version, m.path
FROM memories m
INNER JOIN (
    SELECT name, MAX(created_at) AS max_created_at
    FROM memories
    WHERE deleted = 0
    GROUP BY name
) latest ON m.name = latest.name AND m.created_at = latest.max_created_at
WHERE m.deleted = 0
ORDER BY created_at`
