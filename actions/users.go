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

package actions

import (
	"net/mail"
	"strings"
	"unicode"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

func genUsername(email string) (toName string) {
	if n := strings.Index(email, "+"); n > 0 {
		toName = email[0:n]
	} else {
		toName = email[0:strings.Index(email, "@")]
	}
	if len(toName) > 1 {
		toNameCopy := strings.Replace(toName, ".", " ", -1)
		toName = ""
		lastRuneIsSpace := true
		for _, runeValue := range toNameCopy {
			if lastRuneIsSpace {
				lastRuneIsSpace = false
				toName += string(unicode.ToTitle(runeValue))
			} else {
				toName += string(runeValue)
			}

			if unicode.IsSpace(runeValue) || unicode.IsPunct(runeValue) || unicode.IsSymbol(runeValue) {
				lastRuneIsSpace = true
			}
		}
	}
	return
}

func SendValidationLink(opts *config.Options, user *happydns.UserAuth) error {
	toName := genUsername(user.Email)
	return utils.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Your new account on happyDomain",
		`Welcome to happyDomain!
--------------------

Hi `+toName+`,

We are pleased that you created an account on our great domain name
management platform!

In order to validate your account, please follow this link now:

[Validate my account](`+user.GetRegistrationURL(opts.GetBaseURL())+`)`,
	)
}

func SendRecoveryLink(opts *config.Options, user *happydns.UserAuth) error {
	toName := genUsername(user.Email)
	return utils.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Recover your happyDomain account",
		`Hi `+toName+`,

You've just ask on our platform to recover your account.

In order to define a new password, please follow this link now:

[Recover my account](`+user.GetAccountRecoveryURL(opts.GetBaseURL())+`)`,
	)
}
