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

package happydns

import (
	"time"
)

const (
	LOG_CRIT = iota
	LOG_FATAL
	LOG_ERR2
	LOG_ERR
	LOG_STRONG_WARN
	LOG_WARN
	LOG_WEIRD
	LOG_NACK
	LOG_INFO
	LOG_ACK
	LOG_DEBUG
)

type DomainLog struct {
	// Id is the Log's identifier in the database.
	Id Identifier `json:"id" swaggertype:"string"`

	// IdUser is the identifier of the person responsible for the action.
	IdUser Identifier `json:"id_user" swaggertype:"string"`

	// Date is the date of the action.
	Date time.Time `json:"date"`

	// Content is the description of the action logged.
	Content string `json:"content"`

	// Level reports the criticity level of the action logged.
	Level int8 `json:"level"`
}

type DomainLogWithDomainId struct {
	DomainLog
	DomainId Identifier
}

func NewDomainLog(u *User, level int8, content string) *DomainLog {
	return &DomainLog{
		IdUser:  u.Id,
		Date:    time.Now(),
		Content: content,
		Level:   level,
	}
}

type DomainLogUsecase interface {
	AppendDomainLog(*Domain, *DomainLog) error
	GetDomainLogs(*Domain) ([]*DomainLog, error)
}
