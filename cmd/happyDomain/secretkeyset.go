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
	"fmt"
	"io"

	"git.happydns.org/happyDomain/internal/secret"
)

const secretKeysetUsage = `Usage: happydomain secret-keyset <command> <file>

Manage the instance keyset protecting stored credentials (-secret-keyset-file).

Commands:
  generate <file>  create a new keyset; refuses to overwrite an existing file
  info <file>      list the keys, never their material
`

// runSecretKeyset implements the `happydomain secret-keyset` subcommand and
// returns its exit code.
func runSecretKeyset(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprint(stderr, secretKeysetUsage)
		return 2
	}

	command, path := args[0], args[1]

	switch command {
	case "generate":
		h, err := secret.GenerateInstanceKeyset()
		if err == nil {
			err = secret.WriteNewKeysetFile(path, h)
		}
		if err != nil {
			fmt.Fprintln(stderr, "Unable to generate the keyset:", err)
			return 1
		}
		fmt.Fprintf(stderr, "Keyset written to %s. Back it up: losing it loses every credential it protects.\n", path)

	case "info":
		h, err := secret.ReadKeysetFile(path)
		if err != nil {
			fmt.Fprintln(stderr, "Unable to read the keyset:", err)
			return 1
		}
		for _, k := range secret.DescribeKeyset(h) {
			fmt.Fprintln(stdout, k.String())
		}

	default:
		fmt.Fprint(stderr, secretKeysetUsage)
		return 2
	}

	return 0
}
