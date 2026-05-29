// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	"flag"
	"time"

	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"git.happydns.org/happyDomain/internal/storage"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
)

var (
	path                   string
	blockCacheMiB          int
	writeBufferMiB         int
	openFilesCacheCapacity int
	bloomFilterBits        int
	compactionTableSizeMiB int
	compactionInterval     time.Duration
)

func init() {
	storage.StorageEngines["leveldb"] = Instantiate

	flag.StringVar(&path, "leveldb-path", "happydomain.db", "Path to the LevelDB Database")
	flag.IntVar(&blockCacheMiB, "leveldb-block-cache", 64, "LevelDB block cache capacity, in MiB (goleveldb default: 8)")
	flag.IntVar(&writeBufferMiB, "leveldb-write-buffer", 32, "LevelDB write buffer size, in MiB (goleveldb default: 4)")
	flag.IntVar(&openFilesCacheCapacity, "leveldb-open-files-cache", 4096, "LevelDB open files cache capacity (goleveldb default: 500)")
	flag.IntVar(&bloomFilterBits, "leveldb-bloom-filter-bits", 10, "Bits per key for the LevelDB Bloom filter; 0 disables it (recommended: 10)")
	flag.IntVar(&compactionTableSizeMiB, "leveldb-compaction-table-size", 8, "LevelDB compaction table size, in MiB (goleveldb default: 2)")
	flag.DurationVar(&compactionInterval, "leveldb-compaction-interval", 24*time.Hour, "How often to compact the whole LevelDB keyspace to reclaim space from deleted keys; 0 disables")
}

// options builds the goleveldb tuning options from the configured flags.
func options() *opt.Options {
	o := &opt.Options{
		BlockCacheCapacity:     blockCacheMiB * opt.MiB,
		WriteBuffer:            writeBufferMiB * opt.MiB,
		OpenFilesCacheCapacity: openFilesCacheCapacity,
		CompactionTableSize:    compactionTableSizeMiB * opt.MiB,
	}

	if bloomFilterBits > 0 {
		o.Filter = filter.NewBloomFilter(bloomFilterBits)
	}

	return o
}

func Instantiate() (storage.Storage, error) {
	db, err := NewLevelDBStorage(path, options())
	if err != nil {
		return nil, err
	}

	db.StartCompactionWorker(compactionInterval)

	return kv.NewKVDatabase(db)
}
