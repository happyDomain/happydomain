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
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	happydns "git.happydns.org/happyDomain/model"
)

// runAdminHash implements the `happydomain admin-hash` subcommand. It reads a
// password and prints a bcrypt hash suitable for the
// HAPPYDOMAIN_ADMIN_PASSWORD_HASH environment variable / -admin-password-hash
// flag. On a terminal the password is prompted twice without echo; when stdin
// is piped (scripts, Docker) a single line is read. Prompts go to stderr so the
// hash on stdout can be captured cleanly.
func runAdminHash() {
	pw := readAdminPassword()

	if len(pw) < 8 || len(pw) > 72 {
		fmt.Fprintln(os.Stderr, "Password must be between 8 and 72 bytes long.")
		os.Exit(1)
	}

	// Share the cost with user passwords, so raising it there also strengthens
	// admin hashes generated from here.
	hash, err := bcrypt.GenerateFromPassword(pw, happydns.BcryptCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to hash password:", err)
		os.Exit(1)
	}

	fmt.Println(string(hash))
}

func readAdminPassword() []byte {
	fd := int(os.Stdin.Fd())

	if !term.IsTerminal(fd) {
		// Non-interactive: read a single line from stdin.
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			fmt.Fprintln(os.Stderr, "Unable to read password from stdin.")
			os.Exit(1)
		}
		return []byte(scanner.Text())
	}

	pw1 := promptPassword("Admin password: ")
	pw2 := promptPassword("Confirm password: ")
	if string(pw1) != string(pw2) {
		fmt.Fprintln(os.Stderr, "Passwords do not match.")
		os.Exit(1)
	}
	return pw1
}

func promptPassword(prompt string) []byte {
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read password:", err)
		os.Exit(1)
	}
	return pw
}
