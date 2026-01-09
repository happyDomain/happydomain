// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type PostgreSQLStorage struct {
	db    *sql.DB
	table string
}

// NewPostgreSQLStorage establishes the connection to the PostgreSQL database
func NewPostgreSQLStorage(cfg *PostgreSQLConfig) (s *PostgreSQLStorage, err error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL server: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Printf("Connected to PostgreSQL database: %s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)

	s = &PostgreSQLStorage{
		db:    db,
		table: cfg.Table,
	}

	// Initialize database schema
	if err = s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// initSchema creates the table and index if they don't exist
func (s *PostgreSQLStorage) initSchema() error {
	// Create table with JSONB column
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			key TEXT PRIMARY KEY,
			data JSONB NOT NULL
		)
	`, s.table)

	_, err := s.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create index for prefix searches
	createIndexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_key_prefix
		ON %s (key text_pattern_ops)
	`, s.table, s.table)

	_, err = s.db.Exec(createIndexSQL)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	log.Printf("PostgreSQL schema initialized successfully (table: %s)", s.table)
	return nil
}

func (s *PostgreSQLStorage) Close() error {
	if s.db != nil {
		log.Println("Closing PostgreSQL connection...")
		return s.db.Close()
	}
	return nil
}

func (s *PostgreSQLStorage) DecodeData(data interface{}, v interface{}) error {
	var bytes []byte

	switch d := data.(type) {
	case []byte:
		bytes = d
	case string:
		bytes = []byte(d)
	default:
		return fmt.Errorf("data to decode is not in []byte or string format (%T)", data)
	}

	return json.Unmarshal(bytes, v)
}

func (s *PostgreSQLStorage) Has(key string) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE key = $1)", s.table)

	var exists bool
	err := s.db.QueryRow(query, key).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	return exists, nil
}

func (s *PostgreSQLStorage) Get(key string, v interface{}) error {
	query := fmt.Sprintf("SELECT data FROM %s WHERE key = $1", s.table)

	var jsonData []byte
	err := s.db.QueryRow(query, key).Scan(&jsonData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return happydns.ErrNotFound
		}
		return fmt.Errorf("failed to get key %q: %w", key, err)
	}

	return json.Unmarshal(jsonData, v)
}

func (s *PostgreSQLStorage) Put(key string, v interface{}) error {
	// Marshal value to JSON
	jsonData, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Use UPSERT to handle both insert and update
	query := fmt.Sprintf(`
		INSERT INTO %s (key, data)
		VALUES ($1, $2::jsonb)
		ON CONFLICT (key)
		DO UPDATE SET data = EXCLUDED.data
	`, s.table)

	_, err = s.db.Exec(query, key, jsonData)
	if err != nil {
		return fmt.Errorf("failed to put key %q: %w", key, err)
	}

	return nil
}

func (s *PostgreSQLStorage) FindIdentifierKey(prefix string) (key string, id happydns.Identifier, err error) {
	found := true
	for found {
		id, err = happydns.NewRandomIdentifier()
		if err != nil {
			return
		}
		key = fmt.Sprintf("%s%s", prefix, id.String())

		found, err = s.Has(key)
		if err != nil {
			return
		}
	}
	return
}

func (s *PostgreSQLStorage) Delete(key string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE key = $1", s.table)

	_, err := s.db.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete key %q: %w", key, err)
	}

	return nil
}

func (s *PostgreSQLStorage) Search(prefix string) storage.Iterator {
	query := fmt.Sprintf("SELECT key, data FROM %s WHERE key LIKE $1 || '%%' ORDER BY key", s.table)

	rows, err := s.db.Query(query, prefix)
	if err != nil {
		log.Printf("PostgreSQL Search error: %v", err)
		// Return an iterator with the error
		return &PostgreSQLIterator{
			rows:  nil,
			err:   err,
			valid: false,
		}
	}

	return &PostgreSQLIterator{
		rows:  rows,
		valid: false,
	}
}
