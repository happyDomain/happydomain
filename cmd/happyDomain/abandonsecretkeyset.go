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

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

// runAbandonSecretKeyset implements the `happydomain abandon-secret-keyset`
// subcommand and returns its exit code. It takes the configuration of the
// server, to reach its database.
func runAbandonSecretKeyset(args []string) int {
	// The configuration reads the command line from os.Args.
	os.Args = append([]string{os.Args[0]}, args...)
	opts, err := config.ConsolidateConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	open, ok := storage.StorageEngines[opts.StorageEngine]
	if !ok {
		fmt.Fprintf(os.Stderr, "Nonexistent storage engine: %q\n", opts.StorageEngine)
		return 2
	}
	store, err := open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not open the database (is happyDomain still running?):", err)
		return 1
	}
	defer store.Close()

	return abandonSecretKeyset(store, opts.SecretKeysetFile, os.Stdin, os.Stderr)
}

// abandonSecretKeyset deletes the users' keys and the check record of a lost
// instance keyset, once confirmed on stdin.
func abandonSecretKeyset(store secret.KeysetStorage, keyPath string, stdin io.Reader, out io.Writer) int {
	var key *secret.InstanceKey
	if keyPath != "" {
		k, err := secret.LoadInstanceKey(keyPath)
		if err != nil {
			fmt.Fprintf(out, "The configured keyset %s does not load, it is taken as lost: %s\n", keyPath, err)
		} else {
			key = k
		}
	}

	if key != nil {
		opens, err := key.OpensCheckRecord(store)
		if err != nil {
			fmt.Fprintln(out, err)
			return 1
		}
		if opens {
			fmt.Fprintf(out, "Refusing: the configured keyset %s opens the check record. It is not lost, nothing needs to be abandoned.\n", keyPath)
			return 1
		}
	}

	n, err := secret.CountInstanceSafes(store)
	if err != nil {
		fmt.Fprintln(out, "Unable to list the users' keys:", err)
		return 1
	}
	_, err = store.GetSecretCheck()
	if err != nil && !errors.Is(err, happydns.ErrNotFound) {
		fmt.Fprintln(out, "Unable to read the keyset check record:", err)
		return 1
	}
	if n == 0 && err != nil {
		fmt.Fprintln(out, "Nothing to abandon: no credential is sealed under an instance keyset.")
		return 0
	}

	fmt.Fprintf(out, `This deletes the keys of %d users, and the check record of the instance keyset.
The credentials sealed under the lost keyset (DNS providers, notification
channels, checker options) are then lost for good: their owners will have to
enter them again. Only a copy of the keyset can bring them back.

Stop happyDomain and back up the database before going on.

Type abandon to confirm: `, n)

	answer, _ := bufio.NewReader(stdin).ReadString('\n')
	if strings.TrimSpace(answer) != "abandon" {
		fmt.Fprintln(out, "\nAborted, nothing was deleted.")
		return 1
	}

	deleted, err := secret.AbandonInstanceKeyset(store, key)
	if err != nil {
		fmt.Fprintf(out, "Unable to abandon the keyset, after deleting the keys of %d users: %s\n", deleted, err)
		return 1
	}

	fmt.Fprintf(out, `
Deleted the keys of %d users and the keyset check record.

Start happyDomain again, with secret-policy=plaintext and no keyset, or with a
new keyset (happydomain secret-keyset generate) and secret-policy=instance.
/api/secrets/status counts the credentials to enter again as unreadable.
`, deleted)
	return 0
}
