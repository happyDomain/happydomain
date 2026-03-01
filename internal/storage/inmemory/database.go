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

package inmemory

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

// InMemoryStorage implements the Storage interface using in-memory data structures.
type InMemoryStorage struct {
	mu                  sync.Mutex
	data                map[string][]byte // Generic key-value store for KVStorage interface
	authUsers           map[string]*happydns.UserAuth
	authUsersByEmail    map[string]happydns.Identifier
	domains             map[string]*happydns.Domain
	domainLogs          map[string]*happydns.DomainLogWithDomainId
	domainLogsByDomains map[string][]*happydns.Identifier
	providers           map[string]*happydns.ProviderMessage
	sessions            map[string]*happydns.Session
	users               map[string]*happydns.User
	usersByEmail        map[string]*happydns.User
	zones               map[string]*happydns.ZoneMessage
	lastInsightsRun     *time.Time
	lastInsightsID      happydns.Identifier
}

// NewInMemoryStorage creates a new instance of InMemoryStorage.
func NewInMemoryStorage() (*InMemoryStorage, error) {
	return &InMemoryStorage{
		data:                make(map[string][]byte),
		authUsers:           make(map[string]*happydns.UserAuth),
		authUsersByEmail:    make(map[string]happydns.Identifier),
		domains:             make(map[string]*happydns.Domain),
		domainLogs:          make(map[string]*happydns.DomainLogWithDomainId),
		domainLogsByDomains: make(map[string][]*happydns.Identifier),
		providers:           make(map[string]*happydns.ProviderMessage),
		sessions:            make(map[string]*happydns.Session),
		users:               make(map[string]*happydns.User),
		usersByEmail:        make(map[string]*happydns.User),
		zones:               make(map[string]*happydns.ZoneMessage),
	}, nil
}

// SchemaVersion returns the version of the migration currently in use.
func (s *InMemoryStorage) SchemaVersion() int {
	return 0
}

// DoMigration is the first function called.
func (s *InMemoryStorage) MigrateSchema() error {
	log.Println("YOU ARE USING THE inmemory STORAGE: DATA WILL BE LOST ON HAPPYDOMAIN STOP.")
	// No migration needed for in-memory storage.
	return nil
}

// Tidy should optimize the database, looking for orphan records, ...
func (s *InMemoryStorage) Tidy() error {
	// No tidy needed for in-memory storage.
	return nil
}

// Close shutdown the connection with the database and releases all structure.
func (s *InMemoryStorage) Close() error {
	// No connection to close for in-memory storage.
	return nil
}

// DecodeData decodes data from the interface (expected to be []byte) into v.
func (s *InMemoryStorage) DecodeData(data any, v any) error {
	b, ok := data.([]byte)
	if !ok {
		return fmt.Errorf("data to decode are not in []byte format (%T)", data)
	}
	return json.Unmarshal(b, v)
}

// Has checks if a key exists in the storage.
func (s *InMemoryStorage) Has(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.data[key]
	return exists, nil
}

// Get retrieves a value by key and decodes it into v.
func (s *InMemoryStorage) Get(key string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, exists := s.data[key]
	if !exists {
		return happydns.ErrNotFound
	}
	return json.Unmarshal(data, v)
}

// Put stores a value with the given key.
func (s *InMemoryStorage) Put(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = data
	return nil
}

// FindIdentifierKey finds a unique key with the given prefix.
func (s *InMemoryStorage) FindIdentifierKey(prefix string) (key string, id happydns.Identifier, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		id, err = happydns.NewRandomIdentifier()
		if err != nil {
			return
		}
		key = fmt.Sprintf("%s%s", prefix, id.String())

		if _, exists := s.data[key]; !exists {
			return
		}
	}
}

// Delete removes a key from the storage.
func (s *InMemoryStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// Search returns an iterator for all keys with the given prefix.
func (s *InMemoryStorage) Search(prefix string) storage.Iterator {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Collect all matching keys
	var keys []string
	for k := range s.data {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}

	// Sort keys for consistent iteration order
	sort.Strings(keys)

	// Copy data for the iterator to avoid concurrent access issues
	data := make(map[string][]byte, len(keys))
	for _, k := range keys {
		dataCopy := make([]byte, len(s.data[k]))
		copy(dataCopy, s.data[k])
		data[k] = dataCopy
	}

	return NewKVIterator(keys, data)
}
