package memory

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type MemDeletedValue = int

const (
	NOT_DELETED MemDeletedValue = iota
	DELETED
)

type IMemoryStore interface {
	Insert(m MemoryNote) error
	GetAll() ([]MemoryNote, error)
	// Update(m Memory) error
	Search(query string) ([]MemoryNote, error)
	// Tombstone(id string) error
	Close()
}

type MemoryNote struct {
	id string

	Kind    string   `json:"kind"`
	Name    string   `json:"name"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`

	UpdatedAt time.Time
	CreatedAt time.Time
	Path      string

	deleted MemDeletedValue
	version int
}

type FTSMemoryStore struct {
	memoryPath string
	sqlitePath string
	db         *sql.DB
}

func NewFTSMemoryStore(path string) (*FTSMemoryStore, error) {
	if err := os.MkdirAll(path, 0744); err != nil {
		return nil, err
	}

	sqlitePath := filepath.Join(path, "index.db")
	sqliteConn, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return nil, err
	}

	store := &FTSMemoryStore{
		memoryPath: path,
		sqlitePath: sqlitePath,
		db:         sqliteConn,
	}

	if err := store.setupDB(); err != nil {
		return nil, err
	}

	return store, nil
}

func (store *FTSMemoryStore) Insert(m MemoryNote) error {
	m.Path = filepath.Join(store.memoryPath, m.Kind, m.Name+".md")
	if id, err := uuid.NewV7(); err != nil {
		return err
	} else {
		m.id = id.String()
	}
	m.CreatedAt = time.Now()
	m.version = 1

	if err := store.writeMarkdownFile(m); err != nil {
		return err
	}

	if err := store.insertToDB(m); err != nil {
		return err
	}

	return nil
}

func (store *FTSMemoryStore) Search(q string) ([]MemoryNote, error) {
	return store.searchDB(q)
}

func (store *FTSMemoryStore) Close() {
	store.db.Close()
}

func (store *FTSMemoryStore) GetAll() ([]MemoryNote, error) {
	rows, err := store.db.Query("SELECT id, kind, name, content, tags, created_at, updated_at, deleted, version, path FROM memories WHERE deleted = 0")
	if err != err {
		return nil, err
	}
	defer rows.Close()

	memories := make([]MemoryNote, 0)
	for rows.Next() {
		m := MemoryNote{}
		var tags, path string
		var createdAt, updatedAt, deleted, version int64
		err := rows.Scan(&m.id, &m.Kind, &m.Name, &m.Content, &tags, &createdAt, &updatedAt, &deleted, &version, &path)
		if err != nil {
			slog.Warn("Error while reading memory", "error", err)
		}

		m.Tags = strings.Split(tags, ",")
		m.Path = path
		m.CreatedAt = time.Unix(createdAt, 0)
		m.UpdatedAt = time.Unix(updatedAt, 0)
		m.version = int(version)
		m.deleted = MemDeletedValue(deleted)

		memories = append(memories, m)
	}
	return memories, nil
}

func (store *FTSMemoryStore) writeMarkdownFile(m MemoryNote) error {
	if err := os.MkdirAll(filepath.Dir(m.Path), 0744); err != nil {
		return err
	}
	if err := os.WriteFile(m.Path, []byte(m.Content), 0644); err != nil {
		return err
	}
	return nil
}

func (store *FTSMemoryStore) insertToDB(m MemoryNote) error {
	for i, tag := range m.Tags {
		m.Tags[i] = strings.Trim(tag, " ")
	}

	if _, err := store.db.Exec(INSERT_QUERY,
		m.id,
		m.Kind,
		m.Name,
		m.Content,
		strings.Join(m.Tags, ","),
		m.UpdatedAt.Unix(),
		m.CreatedAt.Unix(),
		m.deleted,
		m.version,
		m.Path,
	); err != nil {
		return err
	} else {
		return nil
	}
}

func (store *FTSMemoryStore) setupDB() error {
	if _, err := store.db.Exec(CREATE_METADATA_TABLE); err != nil {
		return err
	}

	if _, err := store.db.Exec(CREATE_VIRTUAL_TABLE); err != nil {
		return err
	}

	if _, err := store.db.Exec(INSERT_TRIGGER); err != nil {
		return err
	}

	if _, err := store.db.Exec(UPDATE_TRIGGER); err != nil {
		return err
	}

	if _, err := store.db.Exec(DELETE_TRIGGER); err != nil {
		return err
	}
	return nil
}

func (store *FTSMemoryStore) searchDB(q string) ([]MemoryNote, error) {
	rows, err := store.db.Query(SEARCH_QUERY, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := make([]MemoryNote, 0)
	var rank float64
	var tags string
	for rows.Next() {
		var m MemoryNote
		if err := rows.Scan(&m.id, &m.Kind, &m.Name, &m.Content, &tags, &rank); err != nil {
			return nil, err
		} else {
			m.Tags = strings.Split(tags, ",")
			memories = append(memories, m)
		}
	}

	return memories, nil
}
