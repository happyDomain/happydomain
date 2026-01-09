// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongoDBIterator struct {
	ctx     context.Context
	cursor  *mongo.Cursor
	err     error
	current *kvDocument
	valid   bool
}

func NewIterator(ctx context.Context, cursor *mongo.Cursor, err error) *MongoDBIterator {
	return &MongoDBIterator{
		ctx:    ctx,
		cursor: cursor,
		err:    err,
		valid:  err == nil,
	}
}

func (it *MongoDBIterator) Next() bool {
	if it.err != nil || it.cursor == nil {
		it.valid = false
		return false
	}

	if it.cursor.Next(it.ctx) {
		var doc kvDocument
		err := it.cursor.Decode(&doc)
		if err != nil {
			it.err = err
			it.valid = false
			return false
		}
		it.current = &doc
		it.valid = true
		return true
	}

	it.valid = false
	return false
}

func (it *MongoDBIterator) Key() string {
	if it.current == nil {
		return ""
	}
	return it.current.Key
}

func (it *MongoDBIterator) Value() interface{} {
	if it.current == nil {
		return []byte{}
	}
	return it.current.Value
}

func (it *MongoDBIterator) Valid() bool {
	return it.valid
}

func (it *MongoDBIterator) Release() {
	if it.cursor != nil {
		it.cursor.Close(it.ctx)
	}
}
