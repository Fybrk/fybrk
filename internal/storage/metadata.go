package storage

import (
	"database/sql"
	"encoding/json"

	"github.com/Fybrk/fybrk/pkg/types"
	_ "modernc.org/sqlite"
)

type MetadataStore struct {
	db *sql.DB
}

func NewMetadataStore(dbPath string) (*MetadataStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store := &MetadataStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}

	return store, nil
}

func (m *MetadataStore) initTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		path TEXT PRIMARY KEY,
		hash BLOB NOT NULL,
		size INTEGER NOT NULL,
		mod_time DATETIME NOT NULL,
		chunks TEXT NOT NULL,
		version INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		profile INTEGER NOT NULL,
		last_seen DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);
	CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen);
	`

	_, err := m.db.Exec(schema)
	return err
}

func (m *MetadataStore) StoreFileMetadata(metadata *types.FileMetadata) error {
	chunksJSON, err := json.Marshal(metadata.Chunks)
	if err != nil {
		return err
	}

	query := `
	INSERT OR REPLACE INTO files (path, hash, size, mod_time, chunks, version)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = m.db.Exec(query,
		metadata.Path,
		metadata.Hash[:],
		metadata.Size,
		metadata.ModTime,
		string(chunksJSON),
		metadata.Version,
	)

	return err
}

func (m *MetadataStore) GetFileMetadata(path string) (*types.FileMetadata, error) {
	query := `SELECT path, hash, size, mod_time, chunks, version FROM files WHERE path = ?`

	row := m.db.QueryRow(query, path)

	var metadata types.FileMetadata
	var hashBytes []byte
	var chunksJSON string

	err := row.Scan(
		&metadata.Path,
		&hashBytes,
		&metadata.Size,
		&metadata.ModTime,
		&chunksJSON,
		&metadata.Version,
	)

	if err != nil {
		return nil, err
	}

	copy(metadata.Hash[:], hashBytes)

	if err := json.Unmarshal([]byte(chunksJSON), &metadata.Chunks); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (m *MetadataStore) ListFiles() ([]*types.FileMetadata, error) {
	query := `SELECT path, hash, size, mod_time, chunks, version FROM files ORDER BY path`

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*types.FileMetadata

	for rows.Next() {
		var metadata types.FileMetadata
		var hashBytes []byte
		var chunksJSON string

		err := rows.Scan(
			&metadata.Path,
			&hashBytes,
			&metadata.Size,
			&metadata.ModTime,
			&chunksJSON,
			&metadata.Version,
		)

		if err != nil {
			return nil, err
		}

		copy(metadata.Hash[:], hashBytes)

		if err := json.Unmarshal([]byte(chunksJSON), &metadata.Chunks); err != nil {
			return nil, err
		}

		files = append(files, &metadata)
	}

	return files, rows.Err()
}

func (m *MetadataStore) DeleteFileMetadata(path string) error {
	query := `DELETE FROM files WHERE path = ?`
	_, err := m.db.Exec(query, path)
	return err
}

func (m *MetadataStore) StoreDevice(device *types.Device) error {
	query := `
	INSERT OR REPLACE INTO devices (id, name, profile, last_seen)
	VALUES (?, ?, ?, ?)
	`

	_, err := m.db.Exec(query,
		device.ID,
		device.Name,
		int(device.Profile),
		device.LastSeen,
	)

	return err
}

func (m *MetadataStore) GetDevice(id string) (*types.Device, error) {
	query := `SELECT id, name, profile, last_seen FROM devices WHERE id = ?`

	row := m.db.QueryRow(query, id)

	var device types.Device
	var profile int

	err := row.Scan(&device.ID, &device.Name, &profile, &device.LastSeen)
	if err != nil {
		return nil, err
	}

	device.Profile = types.DeviceProfile(profile)
	return &device, nil
}

func (m *MetadataStore) Close() error {
	return m.db.Close()
}
