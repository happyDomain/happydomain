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
	"log"

	"github.com/oracle/nosql-go-sdk/nosqldb"
	"github.com/oracle/nosql-go-sdk/nosqldb/types"
)

type Iterator struct {
	firstPassed bool
	n           *NoSQLStorage
	req         *nosqldb.QueryRequest
	res         *nosqldb.QueryResult
	results     []*types.MapValue
	cur_result  int
	err         error
}

func NewIteratorFromRequest(n *NoSQLStorage, req *nosqldb.QueryRequest) *Iterator {
	return &Iterator{
		n:   n,
		req: req,
	}
}

func (i *Iterator) Release() {}

func (i *Iterator) Next() bool {
	i.err = nil

	if i.res == nil {
		if i.firstPassed && i.req.IsDone() {
			return false
		}
		i.firstPassed = true

		i.res, i.err = i.n.client.Query(i.req)
		if i.err != nil {
			log.Println("error in iterator:", i.err.Error())
			return false
		}
		i.results = nil
	}

	if i.results == nil {
		i.results, i.err = i.res.GetResults()
		if i.err != nil {
			log.Println("error in iterator:", i.err.Error())
			return false
		}
		i.cur_result = 0
	} else {
		i.cur_result += 1
	}

	if i.cur_result+1 >= len(i.results) {
		i.res = nil
	}

	return i.cur_result < len(i.results)
}

func (i *Iterator) Valid() bool {
	_, okkey := i.results[i.cur_result].Get("key")
	_, okvalue := i.results[i.cur_result].Get("value")

	return i.err == nil && okkey && okvalue
}

func (i *Iterator) Key() string {
	key, _ := i.results[i.cur_result].Get("key")
	return key.(string)
}

func (i *Iterator) Value() any {
	value, _ := i.results[i.cur_result].Get("value")
	return value
}
