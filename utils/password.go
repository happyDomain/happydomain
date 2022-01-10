// Copyright or © or Copr. happyDNS (2020)
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

package utils

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// GeneratePassword randomly generates a secure 12 chars long password.
func GeneratePassword() (password string, err error) {
	// This will make a 12 chars long password
	b := make([]byte, 9)

	if _, err = rand.Read(b); err != nil {
		return
	}

	password = base64.StdEncoding.EncodeToString(b)

	// Avoid hard to read characters
	for _, i := range [][2]string{
		{"v", "*"}, {"u", "("},
		{"l", "%"}, {"1", "?"},
		{"o", "@"}, {"O", "!"}, {"0", ">"},
		// This one is to avoid problem with openssl
		{"/", "^"},
	} {
		password = strings.Replace(password, i[0], i[1], -1)
	}

	return
}
