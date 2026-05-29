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
	"database/sql"
	"fmt"

	"git.happydns.org/happyDomain/internal/storage"
)

type batchOpKind uint8

const (
	batchPut batchOpKind = iota
	batchDelete
)

type batchOp struct {
	kind batchOpKind
	key  string
	data []byte // only set for batchPut
}

// Batch stages ops and replays them inside a single sql.Tx so the whole set
// is either committed or rolled back together.
type Batch struct {
	s   *PostgreSQLStorage
	ops []batchOp
}

func (s *PostgreSQLStorage) NewBatch() storage.Batch {
	return &Batch{s: s}
}

func (b *Batch) Put(key string, v any) error {
	data, err := storage.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	b.ops = append(b.ops, batchOp{kind: batchPut, key: key, data: data})
	return nil
}

func (b *Batch) Delete(key string) {
	b.ops = append(b.ops, batchOp{kind: batchDelete, key: key})
}

func (b *Batch) Commit() error {
	if len(b.ops) == 0 {
		return nil
	}

	tx, err := b.s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Rollback is a no-op after a successful Commit.
	defer tx.Rollback()

	// Prepare each statement at most once per Commit and reuse it across
	// every staged op; a fresh tx.Exec would otherwise re-parse and re-plan
	// the same SQL for each row, dominating cost on large migrations.
	var putStmt, delStmt *sql.Stmt
	defer func() {
		if putStmt != nil {
			putStmt.Close()
		}
		if delStmt != nil {
			delStmt.Close()
		}
	}()

	for _, op := range b.ops {
		switch op.kind {
		case batchPut:
			if putStmt == nil {
				putStmt, err = tx.Prepare(fmt.Sprintf(`
					INSERT INTO %s (key, data)
					VALUES ($1, $2::jsonb)
					ON CONFLICT (key)
					DO UPDATE SET data = EXCLUDED.data
				`, b.s.table))
				if err != nil {
					return fmt.Errorf("batch prepare put: %w", err)
				}
			}
			if _, err := putStmt.Exec(op.key, op.data); err != nil {
				return fmt.Errorf("batch put %q: %w", op.key, err)
			}
		case batchDelete:
			if delStmt == nil {
				delStmt, err = tx.Prepare(fmt.Sprintf("DELETE FROM %s WHERE key = $1", b.s.table))
				if err != nil {
					return fmt.Errorf("batch prepare delete: %w", err)
				}
			}
			if _, err := delStmt.Exec(op.key); err != nil {
				return fmt.Errorf("batch delete %q: %w", op.key, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
