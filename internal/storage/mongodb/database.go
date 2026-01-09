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

package database // import "git.happydns.org/happyDomain/internal/storage/mongodb"

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type MongoDBStorage struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	ctx        context.Context
}

// kvDocument represents a key-value pair document in MongoDB
type kvDocument struct {
	Key   string `bson:"_id"`
	Value []byte `bson:"value"`
}

// NewMongoDBStorage establishes the connection to the MongoDB database
func NewMongoDBStorage(uri, dbName string) (s *MongoDBStorage, err error) {
	ctx := context.Background()

	// Set client options
	clientOptions := options.Client().ApplyURI(uri)
	clientOptions.SetTimeout(10 * time.Second)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(dbName)
	collection := db.Collection("kv")

	s = &MongoDBStorage{
		client:     client,
		db:         db,
		collection: collection,
		ctx:        ctx,
	}

	return s, nil
}

func (s *MongoDBStorage) Close() error {
	if s.client != nil {
		return s.client.Disconnect(s.ctx)
	}
	return nil
}

func decodeData(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (s *MongoDBStorage) DecodeData(data interface{}, v interface{}) error {
	b, ok := data.([]byte)
	if !ok {
		return fmt.Errorf("data to decode are not in []byte format (%T)", data)
	}
	return decodeData(b, v)
}

func (s *MongoDBStorage) Has(key string) (bool, error) {
	filter := bson.M{"_id": key}
	count, err := s.collection.CountDocuments(s.ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *MongoDBStorage) Get(key string, v interface{}) error {
	filter := bson.M{"_id": key}
	var doc kvDocument

	err := s.collection.FindOne(s.ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return happydns.ErrNotFound
		}
		return err
	}

	return decodeData(doc.Value, v)
}

func (s *MongoDBStorage) Put(key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": key}
	update := bson.M{"$set": bson.M{"value": data}}
	opts := options.Update().SetUpsert(true)

	_, err = s.collection.UpdateOne(s.ctx, filter, update, opts)
	return err
}

func (s *MongoDBStorage) FindIdentifierKey(prefix string) (key string, id happydns.Identifier, err error) {
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

func (s *MongoDBStorage) Delete(key string) error {
	filter := bson.M{"_id": key}
	_, err := s.collection.DeleteOne(s.ctx, filter)
	return err
}

func (s *MongoDBStorage) Search(prefix string) storage.Iterator {
	// Create a filter that matches all keys starting with the prefix
	filter := bson.M{
		"_id": bson.M{
			"$regex":   "^" + prefix,
			"$options": "",
		},
	}

	cursor, err := s.collection.Find(s.ctx, filter)
	return NewIterator(s.ctx, cursor, err)
}
