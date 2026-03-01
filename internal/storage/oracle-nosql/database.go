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
	"encoding/json"
	"fmt"

	"github.com/oracle/nosql-go-sdk/nosqldb"
	"github.com/oracle/nosql-go-sdk/nosqldb/auth/iam"
	"github.com/oracle/nosql-go-sdk/nosqldb/common"
	"github.com/oracle/nosql-go-sdk/nosqldb/jsonutil"
	"github.com/oracle/nosql-go-sdk/nosqldb/types"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type NoSQLStorage struct {
	client *nosqldb.Client
	config *nosqldb.Config
	table  string
}

// NewOCINoSQLStorage establishes the connection to the database
func NewOCINoSQLStorage(cfg *OCINoSQLConfig) (s *NoSQLStorage, err error) {
	privateKey, err := cfg.privateKey()
	if err != nil {
		return nil, err
	}

	// Create IAM authentication provider
	authProvider, err := iam.NewRawSignatureProvider(cfg.tenancy, cfg.user, cfg.Region, cfg.fingerprint, cfg.compartment, privateKey, &cfg.privateKeyPassphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth provider: %w", err)
	}

	// Create configuration
	config := nosqldb.Config{
		Mode:                  "cloud",
		Region:                common.Region(cfg.Region),
		AuthorizationProvider: authProvider,
	}

	// Create client
	client, err := nosqldb.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create NoSQL client: %w", err)
	}

	return &NoSQLStorage{
		client: client,
		config: &config,
		table:  cfg.Table,
	}, nil
}

func (s *NoSQLStorage) Close() error {
	return s.client.Close()
}

func (n *NoSQLStorage) DecodeData(data any, v any) error {
	return json.Unmarshal([]byte(jsonutil.AsJSON(data)), v)
}

func (n *NoSQLStorage) Get(key string, v any) error {
	gkey := &types.MapValue{}
	gkey.Put("key", key)

	req := &nosqldb.GetRequest{
		TableName: n.table,
		Key:       gkey,
	}

	res, err := n.client.Get(req)
	if err != nil {
		return fmt.Errorf("failed to get key %q: %w", key, err)
	}

	if res.Value == nil {
		return happydns.ErrNotFound
	}

	data, ok := res.Value.Get("value")
	if !ok {
		return fmt.Errorf("unable to find value for the given key")
	}

	return n.DecodeData(data, v)
}

func (n *NoSQLStorage) Put(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("unable to marshal data: %w", err)
	}

	val, err := types.NewMapValueFromJSON(string(data))
	if err != nil {
		return fmt.Errorf("unable to create mapvalue from data: %w", err)
	}

	value := &types.MapValue{}
	value.Put("key", key)
	value.Put("value", val)

	req := &nosqldb.PutRequest{
		TableName: n.table,
		Value:     value,
	}

	_, err = n.client.Put(req)
	if err != nil {
		return fmt.Errorf("failed to update user %q: %w", key, err)
	}

	return nil
}

func (n *NoSQLStorage) Has(key string) (exists bool, err error) {
	gkey := &types.MapValue{}
	gkey.Put("key", key)

	req := &nosqldb.GetRequest{
		TableName: n.table,
		Key:       gkey,
	}

	var res *nosqldb.GetResult
	res, err = n.client.Get(req)
	if err != nil {
		return
	}

	return res.RowExists(), nil
}

func (n *NoSQLStorage) FindIdentifierKey(prefix string) (key string, id happydns.Identifier, err error) {
	found := true
	for found {
		id, err = happydns.NewRandomIdentifier()
		if err != nil {
			return
		}
		key = fmt.Sprintf("%s%s", prefix, id.String())

		found, err = n.Has(key)
		if err != nil {
			return
		}
	}
	return
}

func (n *NoSQLStorage) Delete(key string) error {
	dkey := &types.MapValue{}
	dkey.Put("key", key)

	req := &nosqldb.DeleteRequest{
		TableName: n.table,
		Key:       dkey,
	}

	_, err := n.client.Delete(req)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	return nil
}

func (n *NoSQLStorage) Search(prefix string) storage.Iterator {
	query := fmt.Sprintf("SELECT * FROM %s WHERE regex_like(key, '%s.*')", n.table, prefix)

	return NewIteratorFromRequest(n, &nosqldb.QueryRequest{
		Statement: query,
	})
}
