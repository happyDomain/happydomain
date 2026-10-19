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
	"context"
	"testing"

	"git.happydns.org/happyDomain/internal/dnschecker"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
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

// Restoring, the placeholders an export carries stand for no value and are
// dropped, secret options in clear are sealed under the current policy, and
// sealed ones are kept as they are.
func TestRestoreSealsCheckerOptions(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	owner := happydns.Identifier{0xaa}
	domain := happydns.Identifier{0xbb}
	if err := db.CreateOrUpdateUser(&happydns.User{Id: owner, Email: "owner@example.com"}); err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	secrets, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})

	const alreadySealed = "hds:1:AQ:c2VhbGVk"
	in := &happydns.Backup{CheckerConfigurations: []*happydns.CheckerOptionsPositional{
		{
			CheckName: backupSecretChecker,
			UserId:    &owner,
			Options:   happydns.CheckerOptions{"token": "clear-token", "plain": happydns.RedactedSecret},
		},
		{
			CheckName: backupSecretChecker,
			UserId:    &owner,
			DomainId:  &domain,
			Options:   happydns.CheckerOptions{"token": alreadySealed},
		},
	}}
	if err := backup.NewUsecase(db, secrets).Restore(context.Background(), in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	pos, err := db.GetCheckerConfiguration(backupSecretChecker, &owner, &domain, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, p := range pos {
		switch {
		case p.UserId == nil:
			continue
		case p.DomainId == nil:
			found++
			tok, _ := p.Options["token"].(string)
			if !secret.IsSealed(tok) {
				t.Fatalf("restored token = %q, want it sealed", tok)
			}
			clear, err := secrets.OpenValue(ctx, checkerUC.OptionsSecretContext(backupSecretChecker, &owner, nil, nil, "token"), tok)
			if err != nil || clear != "clear-token" {
				t.Errorf("restored token opens to %q, %v; want %q", clear, err, "clear-token")
			}
			if _, ok := p.Options["plain"]; ok {
				t.Errorf("restored plain = %v, want the placeholder dropped", p.Options["plain"])
			}
		default:
			found++
			if p.Options["token"] != alreadySealed {
				t.Errorf("restored sealed token = %v, want it kept as is", p.Options["token"])
			}
		}
	}
	if found != 2 {
		t.Errorf("found %d of the 2 restored scopes", found)
	}
}

// A configuration whose secret option cannot be sealed is reported and left
// out: the others are restored all the same.
func TestRestoreReportsUnsealableCheckerOptions(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	owner := happydns.Identifier{0xaa}
	domain := happydns.Identifier{0xbb}

	in := &happydns.Backup{CheckerConfigurations: []*happydns.CheckerOptionsPositional{
		{CheckName: backupSecretChecker, UserId: &owner, Options: happydns.CheckerOptions{"token": 42}},
		{CheckName: backupSecretChecker, UserId: &owner, DomainId: &domain, Options: happydns.CheckerOptions{"plain": "kept"}},
	}}
	if err := backup.NewUsecase(db, plaintextSecrets(t)).Restore(context.Background(), in); err == nil {
		t.Error("Restore of a non-string secret option succeeded, want an error")
	}

	pos, err := db.GetCheckerConfiguration(backupSecretChecker, &owner, &domain, nil)
	if err != nil {
		t.Fatal(err)
	}
	restored := false
	for _, p := range pos {
		if p.UserId != nil && p.DomainId == nil {
			t.Errorf("the unsealable configuration was stored: %v", p.Options)
		}
		if p.DomainId != nil && p.Options["plain"] == "kept" {
			restored = true
		}
	}
	if !restored {
		t.Error("the valid configuration was not restored")
	}
}
