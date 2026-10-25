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
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// The templates of a user are indexed under them, the global ones under an
// index of their own.
const (
	zonetplPrimaryPrefix = "zonetpl|"
	zonetplUserPrefix    = "zonetpl-user|"
	zonetplGlobalPrefix  = "zonetpl-global|"
)

func zonetplPrimaryKey(id happydns.Identifier) string {
	return zonetplPrimaryPrefix + id.String()
}

func zonetplUserKey(userId, templateId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", zonetplUserPrefix, userId.String(), templateId.String())
}

func zonetplGlobalKey(templateId happydns.Identifier) string {
	return zonetplGlobalPrefix + templateId.String()
}

// zonetplIndexKey is the index entry of a template, under its owner or among
// the global ones.
func zonetplIndexKey(meta *happydns.ZoneTemplateMeta) string {
	if meta.IsGlobal() {
		return zonetplGlobalKey(meta.Id)
	}
	return zonetplUserKey(*meta.UserId, meta.Id)
}

func (s *KVStorage) ListAllZoneTemplates() (happydns.Iterator[happydns.ZoneTemplateMessage], error) {
	iter := s.db.Search(zonetplPrimaryPrefix)
	return NewKVIterator[happydns.ZoneTemplateMessage](s.db, iter), nil
}

func (s *KVStorage) ListZoneTemplatesByUser(userId happydns.Identifier) ([]*happydns.ZoneTemplateMessage, error) {
	return listByIndex(s, zonetplUserPrefix+userId.String()+"|", s.GetZoneTemplate)
}

func (s *KVStorage) ListGlobalZoneTemplates() ([]*happydns.ZoneTemplateMessage, error) {
	return listByIndex(s, zonetplGlobalPrefix, s.GetZoneTemplate)
}

func (s *KVStorage) GetZoneTemplate(id happydns.Identifier) (*happydns.ZoneTemplateMessage, error) {
	t := &happydns.ZoneTemplateMessage{}
	err := s.db.Get(zonetplPrimaryKey(id), t)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrZoneTemplateNotFound
	}
	return t, err
}

func (s *KVStorage) CreateZoneTemplate(t *happydns.ZoneTemplate) error {
	key, id, err := s.db.FindIdentifierKey(zonetplPrimaryPrefix)
	if err != nil {
		return err
	}
	t.Id = id

	batch := s.db.NewBatch()
	if err := batch.Put(key, t); err != nil {
		return err
	}
	if err := batch.Put(zonetplIndexKey(&t.ZoneTemplateMeta), ""); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) UpdateZoneTemplate(t *happydns.ZoneTemplate) error {
	return s.db.Put(zonetplPrimaryKey(t.Id), t)
}

func (s *KVStorage) DeleteZoneTemplate(id happydns.Identifier) error {
	t, err := s.GetZoneTemplate(id)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	batch.Delete(zonetplIndexKey(&t.ZoneTemplateMeta))
	batch.Delete(zonetplPrimaryKey(id))
	return batch.Commit()
}

func (s *KVStorage) RestoreZoneTemplate(t *happydns.ZoneTemplateMessage) error {
	if t.Id.IsEmpty() {
		return errors.New("a zone template to restore needs its identifier")
	}

	batch := s.db.NewBatch()
	// Stored under another owner, its index entry goes with it.
	if old, err := s.GetZoneTemplate(t.Id); err == nil {
		if oldKey := zonetplIndexKey(&old.ZoneTemplateMeta); oldKey != zonetplIndexKey(&t.ZoneTemplateMeta) {
			batch.Delete(oldKey)
		}
	} else if !errors.Is(err, happydns.ErrZoneTemplateNotFound) {
		return err
	}
	if err := batch.Put(zonetplPrimaryKey(t.Id), t); err != nil {
		return err
	}
	if err := batch.Put(zonetplIndexKey(&t.ZoneTemplateMeta), ""); err != nil {
		return err
	}
	return batch.Commit()
}
