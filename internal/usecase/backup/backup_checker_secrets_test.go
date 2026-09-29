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

package backup_test

import (
	"testing"

	"git.happydns.org/happyDomain/internal/dnschecker"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	happydns "git.happydns.org/happyDomain/model"
)

const backupSecretChecker = "backup_secret_checker"

func init() {
	dnschecker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   backupSecretChecker,
		Name: backupSecretChecker,
		Options: happydns.CheckerOptionsDocumentation{
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "token", Type: "string", Secret: true},
				{Id: "plain", Type: "string"},
			},
		},
	})
}

// The secret checker options of the user leave happyDomain in their export no
// more than their provider credentials do. The administrative backup, which
// Restore depends on, keeps them.
func TestBackupUserRedactsCheckerSecrets(t *testing.T) {
	db, user := seed(t)
	if err := db.UpdateCheckerConfiguration(backupSecretChecker, &user.Id, nil, nil, happydns.CheckerOptions{
		"token": "clear-token",
		"plain": "visible",
	}); err != nil {
		t.Fatal(err)
	}

	uc := backup.NewUsecase(db, plaintextSecrets(t))
	ret := uc.BackupUser(user)
	if len(ret.Errors) > 0 {
		t.Fatalf("unexpected export errors: %v", ret.Errors)
	}
	if len(ret.CheckerConfigurations) != 1 {
		t.Fatalf("exported %d checker configurations, want 1", len(ret.CheckerConfigurations))
	}
	opts := ret.CheckerConfigurations[0].Options
	if opts["token"] != happydns.RedactedSecret {
		t.Errorf("token = %v, want it redacted", opts["token"])
	}
	if opts["plain"] != "visible" {
		t.Errorf("plain = %v, want it exported as is", opts["plain"])
	}

	stored, err := db.GetCheckerConfiguration(backupSecretChecker, &user.Id, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range stored {
		if p.UserId != nil && p.DomainId == nil && p.Options["token"] != "clear-token" {
			t.Errorf("stored token = %v; the export must not touch what is stored", p.Options["token"])
		}
	}

	full := uc.Backup()
	for _, cfg := range full.CheckerConfigurations {
		if cfg.CheckName == backupSecretChecker && cfg.Options["token"] != "clear-token" {
			t.Errorf("administrative backup token = %v, want it kept", cfg.Options["token"])
		}
	}
}

// Options of a checker this build does not load cannot be told secret or not:
// the export withholds all of them rather than risk a credential in clear.
func TestBackupUserRedactsOptionsOfUnknownChecker(t *testing.T) {
	db, user := seed(t)
	if err := db.UpdateCheckerConfiguration("checker_not_loaded", &user.Id, nil, nil, happydns.CheckerOptions{
		"api_key": "clear-key",
	}); err != nil {
		t.Fatal(err)
	}

	ret := backup.NewUsecase(db, plaintextSecrets(t)).BackupUser(user)
	if len(ret.CheckerConfigurations) != 1 {
		t.Fatalf("exported %d checker configurations, want 1", len(ret.CheckerConfigurations))
	}
	if v := ret.CheckerConfigurations[0].Options["api_key"]; v != happydns.RedactedSecret {
		t.Errorf("api_key = %v, want it redacted", v)
	}
}
