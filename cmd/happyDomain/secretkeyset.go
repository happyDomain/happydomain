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
	"strconv"

	"git.happydns.org/happyDomain/internal/secret"
)

const secretKeysetUsage = `Usage: happydomain secret-keyset <command> <file> [key-id]

Manage the instance keyset protecting stored credentials (-secret-keyset-file).

Commands:
  generate <file>  create a new keyset; refuses to overwrite an existing file
  rotate <file>    add a new primary key; previous keys keep opening what they sealed
  info <file>      list the keys, never their material
  delete-key <file> <key-id>
                   remove a key that is no longer needed (see the Secrets page
                   of the admin interface); what it sealed no longer opens
`

// runSecretKeyset implements the `happydomain secret-keyset` subcommand and
// returns its exit code.
func runSecretKeyset(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || len(args) > 3 || (len(args) == 3) != (args[0] == "delete-key") {
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

	case "rotate":
		h, err := secret.ReadKeysetFile(path)
		if err == nil {
			h, err = secret.RotateInstanceKeyset(h)
		}
		if err == nil {
			err = secret.ReplaceKeysetFile(path, h)
		}
		if err != nil {
			fmt.Fprintln(stderr, "Unable to rotate the keyset:", err)
			return 1
		}
		fmt.Fprintln(stderr, "New primary key added. Back the keyset up again, restart happyDomain, then rewrap the safes from the admin interface.")

	case "delete-key":
		id, err := strconv.ParseUint(args[2], 10, 32)
		if err != nil {
			fmt.Fprintf(stderr, "Invalid key id %q: see `happydomain secret-keyset info`.\n", args[2])
			return 2
		}
		h, err := secret.ReadKeysetFile(path)
		if err == nil {
			h, err = secret.DeleteInstanceKey(h, uint32(id))
		}
		if err == nil {
			err = secret.ReplaceKeysetFile(path, h)
		}
		if err != nil {
			fmt.Fprintln(stderr, "Unable to delete the key:", err)
			return 1
		}
		fmt.Fprintln(stderr, "Key deleted. Keep a copy of the keyset from before, as long as you may restore an older backup, then restart happyDomain.")

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
