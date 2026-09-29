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

package happydns

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// SealedSecretPrefix starts every sealed value. A stored value without it is
// plaintext, which keeps records written before sealing existed valid as is.
const SealedSecretPrefix = "hds:1:"

// ErrUnsealedSecret is returned when a Secret that never went through sealing
// is about to be encoded, so a code path that forgets to seal fails instead of
// writing plaintext.
var ErrUnsealedSecret = errors.New("secret has not been sealed")

type secretState uint8

const (
	secretEmpty secretState = iota
	secretClear
	secretSealed
	secretOpened
	secretRedacted
)

// Secret holds a credential entrusted to happyDomain: a provider API key, a
// webhook signing secret...
//
// It is either empty, clear (typed by a user, or a legacy plaintext value read
// from storage), sealed (as read from storage), opened (sealed, with its clear
// value alongside) or redacted (the placeholder the API sends instead of the
// value). See docs/secret-management.md.
//
// Its guarantees are the reason it exists rather than a string:
//   - it refuses to be encoded to JSON while clear, see ErrUnsealedSecret;
//   - it never prints its value through fmt, whatever the verb;
//   - its value is only read through Reveal, where it can be found with grep.
type Secret struct {
	// v is behind a pointer so that fmt, when it falls back to reflection
	// (a bad verb on an enclosing struct disables every Format method), only
	// ever prints an address. Never mutated once set: methods replace it, so
	// copies of a Secret stay independent.
	v *secretValue
}

type secretValue struct {
	state secretState

	// token is what is stored: the sealed value, or the raw value under the
	// plaintext policy. Set when sealed, opened or redacted.
	token string

	// clear is the value in clear. Set when clear or opened. No promise of
	// wiping it from memory is made: Go cannot keep it.
	clear []byte

	// binding tells where an opened secret belongs, as internal/secret
	// encodes it: sealing it again elsewhere must not keep a token bound to
	// another place.
	binding string
}

func (s Secret) state() secretState {
	if s.v == nil {
		return secretEmpty
	}
	return s.v.state
}

// NewSecret returns a clear Secret holding value, or an empty one when value
// is empty.
func NewSecret(value string) Secret {
	if value == "" {
		return Secret{}
	}
	return Secret{&secretValue{state: secretClear, clear: []byte(value)}}
}

// IsEmpty reports whether no value is held.
func (s Secret) IsEmpty() bool { return s.state() == secretEmpty }

// IsZero is IsEmpty, so that `json:",omitzero"` omits an empty Secret.
func (s Secret) IsZero() bool { return s.IsEmpty() }

// IsClear reports whether the value is held in clear and still has to be
// sealed before being stored.
func (s Secret) IsClear() bool { return s.state() == secretClear }

// IsSealed reports whether the value is sealed and has not been opened.
func (s Secret) IsSealed() bool { return s.state() == secretSealed }

// IsOpened reports whether the value is sealed and its clear value known.
func (s Secret) IsOpened() bool { return s.state() == secretOpened }

// IsRedacted reports whether this is the placeholder standing for a stored
// value, see RedactedSecret.
func (s Secret) IsRedacted() bool { return s.state() == secretRedacted }

// Reveal returns the value in clear, or an empty string when it is not known:
// empty, sealed and not opened, or redacted.
//
// Only call it where happyDomain uses the credential on behalf of the user,
// when instantiating a provider for example.
func (s Secret) Reveal() string {
	if s.state() == secretClear || s.state() == secretOpened {
		return string(s.v.clear)
	}
	return ""
}

// Redact turns a non-empty Secret into the placeholder the API sends instead
// of its value. An empty one stays empty: claiming a value is stored for a
// field never filled would make it look set.
func (s *Secret) Redact() {
	if s.state() == secretEmpty {
		return
	}
	*s = Secret{&secretValue{state: secretRedacted, token: RedactedSecret}}
}

// Token returns what is stored for a sealed or opened Secret, and an empty
// string otherwise.
//
// Reserved for internal/secret.
func (s Secret) Token() string {
	if s.state() == secretSealed || s.state() == secretOpened {
		return s.v.token
	}
	return ""
}

// ClearForSealing returns a copy of the value when the Secret is clear, that
// is when it has to be sealed before being stored.
//
// Reserved for internal/secret.
func (s Secret) ClearForSealing() ([]byte, bool) {
	if s.state() != secretClear {
		return nil, false
	}
	return append([]byte(nil), s.v.clear...), true
}

// SetSealed makes the Secret sealed under token, dropping any clear value.
//
// Reserved for internal/secret.
func (s *Secret) SetSealed(token string) {
	*s = Secret{&secretValue{state: secretSealed, token: token}}
}

// SetOpened records that token opens to value where binding tells. The
// plaintext policy uses the raw value as token, so that its records keep their
// current format.
//
// Reserved for internal/secret.
func (s *Secret) SetOpened(token string, value []byte, binding string) {
	*s = Secret{&secretValue{state: secretOpened, token: token, clear: append([]byte(nil), value...), binding: binding}}
}

// Binding returns where an opened Secret was sealed or opened, as given to
// SetOpened, and an empty string for any other state.
//
// Reserved for internal/secret.
func (s Secret) Binding() string {
	if s.state() != secretOpened {
		return ""
	}
	return s.v.binding
}

// MarshalJSON encodes the stored form of the Secret. It fails with
// ErrUnsealedSecret for a clear one.
func (s Secret) MarshalJSON() ([]byte, error) {
	switch s.state() {
	case secretEmpty:
		return []byte(`""`), nil
	case secretSealed:
		if !strings.HasPrefix(s.v.token, SealedSecretPrefix) {
			// Would read back as clear: refuse rather than store it.
			return nil, fmt.Errorf("%w: malformed sealed token", ErrUnsealedSecret)
		}
		return json.Marshal(s.v.token)
	case secretOpened, secretRedacted:
		return json.Marshal(s.v.token)
	default:
		return nil, ErrUnsealedSecret
	}
}

// UnmarshalJSON reads a JSON string or null. A value starting with
// SealedSecretPrefix is sealed, RedactedSecret is redacted, anything else is
// clear.
func (s *Secret) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*s = Secret{}
		return nil
	}

	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("a secret must be a string: %w", err)
	}

	switch {
	case v == "":
		*s = Secret{}
	case v == RedactedSecret:
		*s = Secret{&secretValue{state: secretRedacted, token: RedactedSecret}}
	case strings.HasPrefix(v, SealedSecretPrefix):
		*s = Secret{&secretValue{state: secretSealed, token: v}}
	default:
		*s = NewSecret(v)
	}
	return nil
}

func (s Secret) mask() string {
	if s.state() == secretEmpty {
		return ""
	}
	return RedactedSecret
}

// String never returns the value.
func (s Secret) String() string { return s.mask() }

// GoString never returns the value.
func (s Secret) GoString() string { return fmt.Sprintf("happydns.Secret(%q)", s.mask()) }

// Format never prints the value, whatever the verb.
func (s Secret) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('#') {
		io.WriteString(f, s.GoString())
		return
	}
	io.WriteString(f, s.mask())
}
