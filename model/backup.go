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
	"context"
	"io"
)

// Backup is the document exchanged by the backup endpoints. RestoreReader
// applies the fields in document order, so the field order below carries the
// dependencies between record types: safes before the providers and checker
// options sealed into them, plans before the evaluations referencing them.
type Backup struct {
	Version                  int
	UsersAuth                UserAuths
	Users                    []*User
	Safes                    []*Safe           `json:",omitempty"`
	DefaultKeystores         map[string]string `json:",omitempty"`
	Domains                  []*Domain
	DomainsLogs              map[string][]*DomainLog
	Zones                    []*ZoneMessage
	Providers                []*ProviderMessage
	Sessions                 []*Session
	DomainWatches            []*DomainWatch         `json:",omitempty"`
	ZoneTemplates            []*ZoneTemplateMessage `json:",omitempty"`
	NotificationChannels     []*NotificationChannel
	NotificationPreferences  []*NotificationPreference
	NotificationStates       []*NotificationState
	NotificationRecords      []*NotificationRecord
	CheckerConfigurations    []*CheckerOptionsPositional
	CheckPlans               []*CheckPlan
	CheckEvaluations         []*CheckEvaluation
	Executions               []*Execution
	DiscoveryEntries         []*StoredDiscoveryEntry
	DiscoveryObservationRefs []*DiscoveryObservationRef
	ObservationSnapshots     []*ObservationSnapshot
	Errors                   []string
}

// BackupUsecase orchestrates the export and re-import of every persistent
// resource managed by happyDomain. It is the entry point used by the admin
// API to take or restore a complete dump of the system.
type BackupUsecase interface {
	// WriteBackup streams a complete backup as JSON to w with a memory
	// footprint independent of database size.
	WriteBackup(w io.Writer) error
	// WriteBackupUser streams a single user's data export as JSON to w, with
	// credentials and live session IDs stripped.
	WriteBackupUser(w io.Writer, user *User) error
	// RestoreReader streams a backup from r, applying one record at a time so
	// the whole dataset is never held in memory.
	RestoreReader(ctx context.Context, r io.Reader) error
}
