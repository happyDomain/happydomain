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
	"git.happydns.org/happyDomain/model"
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

// The export of an account withholds the checker options that may be
// secrets, as the API does.
func TestBackupUserWithholdsCheckerOptions(t *testing.T) {
	db, user := seed(t)
	if err := db.UpdateCheckerConfiguration(backupSecretChecker, &user.Id, nil, nil, happydns.CheckerOptions{"token": "s3cr3t", "plain": "visible"}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateCheckerConfiguration("gone_checker", &user.Id, nil, nil, happydns.CheckerOptions{"anything": "unknown"}); err != nil {
		t.Fatal(err)
	}

	ret := backup.NewUsecase(db, plaintextSecrets(t)).BackupUser(user)
	found := 0
	for _, cfg := range ret.CheckerConfigurations {
		switch cfg.CheckName {
		case backupSecretChecker:
			found++
			if cfg.Options["token"] != happydns.RedactedSecret || cfg.Options["plain"] != "visible" {
				t.Errorf("exported options = %v", cfg.Options)
			}
		case "gone_checker":
			found++
			if cfg.Options["anything"] != happydns.RedactedSecret {
				t.Errorf("exported options of an unknown checker = %v", cfg.Options)
			}
		}
	}
	if found != 2 {
		t.Errorf("exported %d of the 2 configurations", found)
	}

	// The stored options are left alone.
	pos, err := db.GetCheckerConfiguration(backupSecretChecker, &user.Id, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pos {
		if p.UserId != nil && p.Options["token"] != "s3cr3t" {
			t.Errorf("stored token = %v after the export", p.Options["token"])
		}
	}
}
