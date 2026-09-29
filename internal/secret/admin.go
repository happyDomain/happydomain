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

package secret

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/tink-crypto/tink-go/v2/keyset"
	tinkpb "github.com/tink-crypto/tink-go/v2/proto/tink_go_proto"
	"github.com/tink-crypto/tink-go/v2/tink"

	"git.happydns.org/happyDomain/model"
)

// Counts tells how the secrets of some objects are stored.
type Counts struct {
	// Clear counts the secrets stored in clear.
	Clear int `json:"clear"`

	// Sealed counts the secrets that open, by kind of safe.
	Sealed map[string]int `json:"sealed"`

	// Unreadable counts the sealed secrets that do not open, and the
	// placeholders stored by mistake in place of a secret.
	Unreadable int `json:"unreadable"`

	// Undecodable counts the objects whose secrets could not be looked at:
	// a record that does not decode, an unknown provider type, a safe that
	// could not be read.
	Undecodable int `json:"undecodable"`

	// Problems names some of the undecodable objects, and why.
	Problems []string `json:"problems,omitempty"`
}

// ResealReport tells what a reseal of every object of a type did.
type ResealReport struct {
	ObjectType string `json:"objectType"`

	// Processed counts the objects looked at.
	Processed int `json:"processed"`

	// Changed counts the objects written back.
	Changed int `json:"changed"`

	// Skipped counts the objects left for a later run: changed meanwhile,
	// or owned by a user that no longer exists.
	Skipped int `json:"skipped"`

	// Failed counts the objects left as they were because of an error.
	Failed int `json:"failed"`

	// Errors names some of the skipped and failed objects, and why.
	Errors []string `json:"errors,omitempty"`
}

// Inspect adds how the secrets of obj are stored to c. It tries to open each
// sealed one, in a copy: obj is left untouched.
func (m *Manager) Inspect(ctx context.Context, sc SecretContext, obj any, c *Counts) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return err
	}
	if c.Sealed == nil {
		c.Sealed = map[string]int{}
	}

	cp, err := clone(obj)
	if err != nil {
		return err
	}

	primitives := map[string]tink.AEAD{}

	return Walk(cp, func(path string, s *happydns.Secret) error {
		fsc := sc
		fsc.Field = path
		return m.inspectOne(fsc, s, primitives, c)
	})
}

func (m *Manager) inspectOne(sc SecretContext, s *happydns.Secret, primitives map[string]tink.AEAD, c *Counts) error {
	if c.Sealed == nil {
		c.Sealed = map[string]int{}
	}

	switch {
	case s.IsEmpty():
		return nil
	case s.IsClear():
		c.Clear++
		return nil
	case !s.IsSealed():
		// A placeholder stored in place of the secret: the secret is lost.
		c.Unreadable++
		return nil
	}

	if err := m.open(sc, s, primitives); errors.Is(err, ErrUnopenable) {
		c.Unreadable++
		return nil
	} else if err != nil {
		// Maybe the storage is down: the value may open a moment later, so
		// it is not counted as lost, and its object is reported as not
		// looked at.
		return err
	}

	// open only accepts instance safes, see safeRegistry.aead.
	c.Sealed[KindInstance]++
	return nil
}

// ResealObject stores the secrets of obj, as read from storage, the way the
// current policy stores new ones: sealing clear values under the instance
// policy, opening sealed ones to store them in clear under the plaintext
// policy. It reports whether obj changed and has to be written back.
//
// A value that will never open, or a placeholder stored by mistake, is lost
// already: it is left as it is, so that it does not keep the other secrets of
// its object from being stored the way the policy does. Any other failure,
// such as a safe that cannot be read for now, fails the whole object, leaving
// obj untouched.
func (m *Manager) ResealObject(ctx context.Context, sc SecretContext, obj any) (bool, error) {
	if m == nil {
		return false, errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return false, err
	}

	cp, err := clone(obj)
	if err != nil {
		return false, err
	}

	// Only what the policy stores differently is worth opening.
	todo := false
	err = Walk(cp, func(_ string, s *happydns.Secret) error {
		switch {
		case s.IsClear():
			todo = todo || m.policy != PolicyPlaintext
		case s.IsSealed():
			todo = todo || m.policy == PolicyPlaintext
		}
		return nil
	})
	if err != nil || !todo {
		return false, err
	}

	x := m.newSealer(sc.Owner)
	x.keepUnopenable = true

	changed := false
	err = transform(cp, sc, func(fsc SecretContext, s *happydns.Secret) error {
		if s.IsRedacted() {
			return nil
		}
		wasClear, wasSealed := s.IsClear(), s.IsSealed()
		if err := x.seal(fsc, s); err != nil {
			return err
		}
		if (wasClear && m.policy != PolicyPlaintext) || (wasSealed && !IsSealed(s.Token())) {
			changed = true
		}
		return nil
	})
	if err != nil || !changed {
		return false, err
	}

	// Only now that everything went through: obj is never left half done.
	src, dst := reflectElem(cp), reflectElem(obj)
	dst.Set(src)
	return true, nil
}

// KeyUsage counts the safes wrapped by each key of the instance keyset, by
// key identifier. A key no safe uses any more can be removed from the keyset.
func (m *Manager) KeyUsage() (map[uint32]int, error) {
	usage, _, err := m.keyUsage()
	return usage, err
}

// keyUsage is KeyUsage, also counting the safes whose wrapping key cannot be
// told: a damaged record must not hide the usage of every other key.
func (m *Manager) keyUsage() (usage map[uint32]int, unreadable int, err error) {
	if m == nil || m.safes == nil {
		return nil, 0, errNoManager
	}

	usage = map[uint32]int{}
	damaged, err := scanSafes(m.safes.store, func(safe *happydns.Safe) bool {
		for _, w := range safe.Keyring {
			if w.KEK != KEKInstance {
				continue
			}
			id, err := wrappingKeyId(w)
			if err != nil {
				unreadable++
				continue
			}
			usage[id]++
		}
		return true
	})
	return usage, unreadable + len(damaged), err
}

// SafeObjectType names safes in the reports.
const SafeObjectType = "safe"

// safeName names a safe in the reports.
func safeName(safe *happydns.Safe) string { return "safe " + safe.Id.String() }

// Rewrap wraps again, under the primary instance key, the key of every safe
// wrapped by another one. The sealed values are not touched. A safe that
// fails is reported and left as it was; run it again to resume.
func (m *Manager) Rewrap(ctx context.Context) (ResealReport, error) {
	if m == nil || m.safes == nil || m.safes.key == nil {
		return ResealReport{ObjectType: SafeObjectType}, errors.New("no instance keyset configured")
	}

	iter, err := m.safes.store.ListAllSafes()
	if err != nil {
		return ResealReport{ObjectType: SafeObjectType}, err
	}

	return ResealAll(SafeObjectType, iter,
		safeName,
		func(safe *happydns.Safe) (bool, error) { return m.rewrapSafe(safe.Id) },
	)
}

// rewrapSafe rewraps the safe id, as stored now, if it needs it. The write is
// conditional: a safe changed or deleted since it was read is left alone.
func (m *Manager) rewrapSafe(id happydns.Identifier) (bool, error) {
	key := m.safes.key

	changed := false
	err := m.safes.store.ReplaceSafe(id, func(safe *happydns.Safe) (*happydns.Safe, error) {
		needed, err := key.needsRewrap(safe)
		if err != nil || !needed {
			return nil, err
		}

		dek, err := key.Unwrap(safe)
		if err != nil {
			return nil, err
		}
		w, err := key.Wrap(safe, dek)
		if err != nil {
			return nil, err
		}

		// The new wrapping takes the place of the first instance entry;
		// any other one goes, they all wrapped the same key.
		keyring := make([]happydns.WrappedKeyset, 0, len(safe.Keyring))
		placed := false
		for _, e := range safe.Keyring {
			if e.KEK != KEKInstance {
				keyring = append(keyring, e)
			} else if !placed {
				keyring = append(keyring, w)
				placed = true
			}
		}
		safe.Keyring = keyring

		changed = true
		return safe, nil
	})
	if errors.Is(err, happydns.ErrSafeNotFound) {
		// Deleted since it was listed.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return changed, nil
}

// needsRewrap tells whether an instance entry of safe is wrapped by another
// key than the primary one.
func (k *InstanceKey) needsRewrap(safe *happydns.Safe) (bool, error) {
	for _, w := range safe.Keyring {
		if w.KEK != KEKInstance {
			continue
		}
		id, err := wrappingKeyId(w)
		if err != nil {
			return false, err
		}
		if id != k.PrimaryKeyId() {
			return true, nil
		}
	}
	return false, nil
}

// checkUnwrappable fails when an instance entry of safe is wrapped by a key
// the keyset does not hold enabled, or by a key that cannot be told. Such a
// safe does not open now, but is not lost: putting the key back, or enabling
// it again, opens it.
func (k *InstanceKey) checkUnwrappable(safe *happydns.Safe) error {
	enabled := map[uint32]bool{}
	for _, info := range k.handle.KeysetInfo().GetKeyInfo() {
		if info.GetStatus() == tinkpb.KeyStatusType_ENABLED {
			enabled[info.GetKeyId()] = true
		}
	}

	for _, w := range safe.Keyring {
		if w.KEK != KEKInstance {
			continue
		}
		id, err := wrappingKeyId(w)
		if err != nil {
			return fmt.Errorf("the instance key wrapping it cannot be told: %w", err)
		}
		if !enabled[id] {
			return fmt.Errorf("wrapped by the instance key %d, missing from the keyset or not enabled: put it back to open what it holds", id)
		}
	}
	return nil
}

// wrappingKeyId returns the identifier of the instance key that wrapped w,
// read from the Tink output prefix of the ciphertext: one version byte, then
// the key identifier.
func wrappingKeyId(w happydns.WrappedKeyset) (uint32, error) {
	enc, err := keyset.NewBinaryReader(bytes.NewReader(w.Blob)).ReadEncrypted()
	if err != nil {
		return 0, err
	}

	return ciphertextKeyId(enc.GetEncryptedKeyset())
}

// KeyStatus describes a key of the instance keyset and how many safes it
// wraps.
type KeyStatus struct {
	KeyInfo

	// Safes counts the safes whose key it wraps.
	Safes int `json:"safes"`
}

// KeyStatus lists the keys of the instance keyset with the number of safes
// each wraps. A key wrapping safes but missing from the keyset is listed
// with the status MISSING: those safes do not open. Safes whose wrapping key
// cannot be told, damaged ones, are counted under the status UNREADABLE.
func (m *Manager) KeyStatus() ([]KeyStatus, error) {
	if m == nil || m.safes == nil {
		return nil, errNoManager
	}

	usage, unreadable, err := m.keyUsage()
	if err != nil {
		return nil, err
	}

	var keys []KeyStatus
	if m.safes.key != nil {
		for _, k := range DescribeKeyset(m.safes.key.handle) {
			keys = append(keys, KeyStatus{KeyInfo: k, Safes: usage[k.Id]})
			delete(usage, k.Id)
		}
	}
	missing := slices.Sorted(maps.Keys(usage))
	for _, id := range missing {
		keys = append(keys, KeyStatus{KeyInfo: KeyInfo{Id: id, Status: "MISSING"}, Safes: usage[id]})
	}
	if unreadable > 0 {
		keys = append(keys, KeyStatus{KeyInfo: KeyInfo{Status: "UNREADABLE"}, Safes: unreadable})
	}
	return keys, nil
}

// Policy is how new secrets are sealed.
func (m *Manager) Policy() Policy {
	if m == nil {
		return ""
	}
	return m.policy
}

// refusePlaceholder refuses the placeholder the API sends in place of a
// secret: stored by mistake, it stands for no credential.
func refusePlaceholder(sc SecretContext, value string) error {
	if value == happydns.RedactedSecret {
		return fmt.Errorf("%s: %w", sc.Field, ErrRedactedSecret)
	}
	return nil
}

// SealValue returns value, a secret stored in a map rather than in a
// struct, sealed under the current policy. A value already sealed is
// returned as is, except under the plaintext policy, which returns it in
// clear when it opens. sc must name the field.
func (m *Manager) SealValue(ctx context.Context, sc SecretContext, value string) (string, error) {
	if err := refusePlaceholder(sc, value); err != nil {
		return "", err
	}
	if IsSealed(value) {
		if m.policy != PolicyPlaintext {
			return value, nil
		}
		// Under the plaintext policy, stored in clear like a new value,
		// as SealObject does.
		clear, err := m.OpenValue(ctx, sc, value)
		if errors.Is(err, ErrUnopenable) {
			// Lost already: kept as is, so that the values stored beside
			// it can still be saved.
			return value, nil
		}
		if err != nil {
			return "", err
		}
		value = clear
	}
	s := happydns.NewSecret(value)
	if err := m.SealSecret(ctx, sc, &s); err != nil {
		return "", err
	}
	return s.Token(), nil
}

// OpenValue returns value, as stored, in clear. sc must name the field.
func (m *Manager) OpenValue(ctx context.Context, sc SecretContext, value string) (string, error) {
	return m.NewValueOpener().Open(ctx, sc, value)
}

// ValueOpener opens several values in a row, such as the options of a
// checker, reading each safe they are sealed in once.
type ValueOpener struct {
	m *Manager

	// primitives opens the sealed values met, by safe identifier.
	primitives map[string]tink.AEAD
}

// NewValueOpener returns a ValueOpener, to drop once the values are opened.
func (m *Manager) NewValueOpener() *ValueOpener {
	return &ValueOpener{m: m, primitives: map[string]tink.AEAD{}}
}

// Open returns value, as stored, in clear. sc must name the field.
func (o *ValueOpener) Open(ctx context.Context, sc SecretContext, value string) (string, error) {
	if err := refusePlaceholder(sc, value); err != nil {
		return "", err
	}
	if !IsSealed(value) {
		return value, nil
	}
	if o.m == nil {
		return "", errNoManager
	}
	if err := sc.Validate(); err != nil {
		return "", err
	}
	s := happydns.ParseSecret(value)
	if err := o.m.open(sc, &s, o.primitives); err != nil {
		return "", err
	}
	return s.Reveal(), nil
}

// InspectValue adds how value, as stored, is protected to c.
func (m *Manager) InspectValue(ctx context.Context, sc SecretContext, value string, c *Counts) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.Validate(); err != nil {
		return err
	}
	s := happydns.ParseSecret(value)
	return m.inspectOne(sc, &s, map[string]tink.AEAD{}, c)
}

// ResealValue returns value, as stored, the way the current policy stores new
// secrets, and whether that differs from what is stored. A value that will
// never open is lost already, and is returned as it is.
func (m *Manager) ResealValue(ctx context.Context, sc SecretContext, value string) (string, bool, error) {
	if m == nil {
		return "", false, errNoManager
	}
	if err := refusePlaceholder(sc, value); err != nil {
		return "", false, err
	}

	sealed := IsSealed(value)
	switch {
	case value == "":
		return value, false, nil
	case sealed && m.policy != PolicyPlaintext, !sealed && m.policy == PolicyPlaintext:
		return value, false, nil
	}

	clear, err := m.OpenValue(ctx, sc, value)
	if errors.Is(err, ErrUnopenable) {
		// Lost already: left as it is.
		return value, false, nil
	}
	if err != nil {
		return "", false, err
	}
	out, err := m.SealValue(ctx, sc, clear)
	if err != nil {
		return "", false, err
	}
	return out, true, nil
}

// ErrSafesLeft is returned by DropInstanceSafes when a safe could not be
// dropped: the keyset is still needed, and must not be removed yet.
var ErrSafesLeft = errors.New("instance safes left, the keyset check record is kept: keep the keyset, deal with them and run it again")

// DropInstanceSafes deletes every instance safe, then the check record of the
// instance keyset, and reports what it did. Once done, the keyset can be
// removed from the configuration. Whatever is still sealed in those safes no
// longer opens: check before that nothing is.
//
// A safe that fails, such as a record that does not decode, or one wrapped by
// a key the keyset lacks, whose values could open again once the key is back,
// is reported and left as it was, and the others are deleted. The check
// record is then kept, keeping startup asking for the keyset, and it fails
// with ErrSafesLeft: the keyset must not be removed yet.
//
// It only runs under the plaintext policy, which never creates a safe, and
// with the keyset still configured. It can be interrupted and run again.
func (m *Manager) DropInstanceSafes(ctx context.Context, checks CheckStorage) (ResealReport, error) {
	report := ResealReport{ObjectType: SafeObjectType}
	if m == nil || m.safes == nil {
		return report, errNoManager
	}
	if m.policy != PolicyPlaintext {
		return report, errors.New("the instance safes can only be dropped under the plaintext policy")
	}
	if m.safes.key == nil {
		return report, errors.New("the instance safes can only be dropped with the instance keyset still configured")
	}

	// Collected before deleting: writing while listing is not safe on every
	// storage.
	var safes []*happydns.Safe
	damaged, err := scanSafes(m.safes.store, func(safe *happydns.Safe) bool {
		if safe.Kind == KindInstance {
			safes = append(safes, safe)
		}
		return true
	})
	if err != nil {
		return report, err
	}

	var probs problems
	for _, key := range damaged {
		report.Processed++
		report.Failed++
		probs.add("%s: does not decode, it may be an instance safe", key)
	}
	for _, safe := range safes {
		report.Processed++
		// What it holds counts as unreadable, but is not lost.
		if err := m.safes.key.checkUnwrappable(safe); err != nil {
			report.Failed++
			probs.add("%s: %s", safeName(safe), err.Error())
			continue
		}
		deleted, err := m.deleteSafe(safe.Id)
		if err != nil {
			report.Failed++
			probs.add("%s: %s", safeName(safe), err.Error())
			continue
		}
		if deleted {
			report.Changed++
		}
	}
	report.Errors = probs.list()

	if report.Failed > 0 {
		return report, fmt.Errorf("%w: %s", ErrSafesLeft, strings.Join(report.Errors, "; "))
	}

	// Last: interrupted before, the keyset is still checked at startup.
	if err := checks.DeleteSecretCheck(); err != nil {
		return report, fmt.Errorf("unable to delete the keyset check record: %w", err)
	}

	return report, nil
}
