// Copyright or © or Copr. happyDNS (2021)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package actions

import (
	"net/mail"
	"strings"
	"unicode"

	"git.happydns.org/happydomain/config"
	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/utils"
)

func genUsername(user *happydns.UserAuth) (toName string) {
	if n := strings.Index(user.Email, "+"); n > 0 {
		toName = user.Email[0:n]
	} else {
		toName = user.Email[0:strings.Index(user.Email, "@")]
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
	toName := genUsername(user)
	return utils.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Your new account on happyDomain",
		`Welcome to happyDomain!
--------------------

Hi `+toName+`,

We are pleased that you created an account on our great domain name
management platform!

In order to validate your account, please follow this link now:

[Validate my account](`+opts.GetRegistrationURL(user)+`)`,
	)
}

func SendRecoveryLink(opts *config.Options, user *happydns.UserAuth) error {
	toName := genUsername(user)
	return utils.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Recover you happyDomain account",
		`Hi `+toName+`,

You've just ask on our platform to recover your account.

In order to define a new password, please follow this link now:

[Recover my account](`+opts.GetAccountRecoveryURL(user)+`)`,
	)
}
