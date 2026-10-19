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

package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// KeystoreUsage is a keystore, with the objects whose secrets it holds.
type KeystoreUsage struct {
	secret.Keystore

	Objects []secret.ObjectSafes `json:"objects"`
}

// KeystoreUsecase lets an owner, or the administrator on their behalf,
// manage the keystores their secrets are sealed in.
type KeystoreUsecase struct {
	manager   *secret.Manager
	holders   map[string]SecretHolder
	passwords LoginPasswords
}

// LoginPasswords tells whether owners log in with a password, and checks it:
// a keystore bound to the login password is only created knowing it.
type LoginPasswords interface {
	HasLoginPassword(owner happydns.Identifier) bool
	CheckLoginPassword(owner happydns.Identifier, password string) error
}

// WithLoginPasswords lets owners logging in with a password create keystores
// bound to it.
func (u *KeystoreUsecase) WithLoginPasswords(passwords LoginPasswords) *KeystoreUsecase {
	u.passwords = passwords
	return u
}

func NewKeystoreUsecase(manager *secret.Manager, holders map[string]SecretHolder) *KeystoreUsecase {
	return &KeystoreUsecase{manager: manager, holders: holders}
}

// keystoreError gives the errors of the keystores their status.
func keystoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, secret.ErrUnknownKeystore):
		return happydns.NotFoundError{Msg: err.Error()}
	case errors.Is(err, secret.ErrInvalidKeystore), errors.Is(err, secret.ErrKeystoreNotUnattended), errors.Is(err, secret.ErrReferencesOnly):
		return happydns.ValidationError{Msg: err.Error()}
	case errors.Is(err, secret.ErrInstanceKeystore), errors.Is(err, secret.ErrWrongUnlockMaterial):
		return happydns.ForbiddenError{Msg: err.Error()}
	}
	return err
}

// objects lists the objects of owner holding sealed secrets, of every type.
func (u *KeystoreUsecase) objects(ctx context.Context, owner happydns.Identifier) ([]secret.ObjectSafes, error) {
	var all []secret.ObjectSafes
	for _, t := range slices.Sorted(maps.Keys(u.holders)) {
		objects, err := u.holders[t].OwnerObjects(ctx, owner)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t, err)
		}
		all = append(all, objects...)
	}
	return all, nil
}

// List lists the keystores of owner, each with the objects it holds secrets
// of.
func (u *KeystoreUsecase) List(ctx context.Context, owner happydns.Identifier) ([]KeystoreUsage, error) {
	keystores, err := u.manager.Keystores(ctx, owner)
	if err != nil {
		return nil, err
	}
	objects, err := u.objects(ctx, owner)
	if err != nil {
		return nil, err
	}

	list := make([]KeystoreUsage, 0, len(keystores))
	for _, ks := range keystores {
		usage := KeystoreUsage{Keystore: ks, Objects: []secret.ObjectSafes{}}
		if !ks.SafeId.IsEmpty() {
			for _, o := range objects {
				if slices.ContainsFunc(o.Safes, ks.SafeId.Equals) {
					usage.Objects = append(usage.Objects, o)
				}
			}
		}
		list = append(list, usage)
	}
	return list, nil
}

// hasLoginPassword tells whether owner logs in with a password.
func (u *KeystoreUsecase) hasLoginPassword(owner happydns.Identifier) bool {
	return u.passwords != nil && u.passwords.HasLoginPassword(owner)
}

// Kinds lists the kinds of personal keystore owner may create: those bound
// to the login password only when they have one.
func (u *KeystoreUsecase) Kinds(owner happydns.Identifier) []secret.KindInfo {
	kinds := []secret.KindInfo{}
	for _, k := range u.manager.CreatableKinds() {
		if !k.PasswordBound || u.hasLoginPassword(owner) {
			kinds = append(kinds, k)
		}
	}
	return kinds
}

// Create creates a personal keystore of kind for owner. One bound to the
// login password needs it, checked here.
func (u *KeystoreUsecase) Create(ctx context.Context, owner happydns.Identifier, kind, name string, params json.RawMessage) (secret.Keystore, error) {
	for _, k := range u.manager.CreatableKinds() {
		if k.Kind != kind || !k.PasswordBound {
			continue
		}
		if !u.hasLoginPassword(owner) {
			return secret.Keystore{}, happydns.ValidationError{Msg: fmt.Sprintf("a keystore of kind %q opens with the login password: this account has none", kind)}
		}
		var p map[string]string
		// Params that do not decode give an empty password, refused below.
		_ = json.Unmarshal(params, &p)
		if err := u.passwords.CheckLoginPassword(owner, p[secret.PasswordParam]); err != nil {
			return secret.Keystore{}, happydns.ForbiddenError{Msg: "this is not the login password of the account"}
		}
	}

	ks, err := u.manager.CreateKeystore(ctx, owner, kind, name, params)
	return ks, keystoreError(err)
}

// Unlock opens ref, a keystore of owner, with material, for the session of
// ctx.
func (u *KeystoreUsecase) Unlock(ctx context.Context, owner happydns.Identifier, ref string, material json.RawMessage) error {
	return keystoreError(u.manager.UnlockKeystore(ctx, owner, ref, material))
}

// Lock closes ref, a keystore of owner, for the session of ctx.
func (u *KeystoreUsecase) Lock(ctx context.Context, owner happydns.Identifier, ref string) error {
	return keystoreError(u.manager.LockKeystore(ctx, owner, ref))
}

// UnlockAtLogin opens, for the session of ctx, the keystores of owner bound
// to password, their login password.
func (u *KeystoreUsecase) UnlockAtLogin(ctx context.Context, owner happydns.Identifier, password string) {
	u.manager.UnlockAtLogin(ctx, owner, password)
}

// LockSession closes every keystore unlocked during scope, a session ending.
func (u *KeystoreUsecase) LockSession(scope string) {
	u.manager.LockScope(scope)
}

// PasswordChanged binds the keystores of owner bound to old, their previous
// login password, to new.
func (u *KeystoreUsecase) PasswordChanged(ctx context.Context, owner happydns.Identifier, old, new string) error {
	return u.manager.PasswordChanged(ctx, owner, old, new)
}

// Delete deletes ref, a keystore of owner, once it holds no secret: they are
// migrated first. A keystore of the instance is only deleted by the
// administrator, instance set.
func (u *KeystoreUsecase) Delete(ctx context.Context, owner happydns.Identifier, ref string, instance bool) error {
	if !instance {
		keystores, err := u.manager.Keystores(ctx, owner)
		if err != nil {
			return err
		}
		for _, ks := range keystores {
			if ks.Ref == ref && ks.Scope == secret.ScopeInstance {
				return happydns.ForbiddenError{Msg: secret.ErrInstanceKeystore.Error()}
			}
		}
	}

	id, err := u.manager.KeystoreSafe(ctx, owner, ref)
	if errors.Is(err, happydns.ErrSafeNotFound) {
		return happydns.NotFoundError{Msg: fmt.Sprintf("keystore %q is not created yet", ref)}
	}
	if err != nil {
		return keystoreError(err)
	}

	objects, err := u.objects(ctx, owner)
	if err != nil {
		return err
	}
	var used []string
	for _, o := range objects {
		if slices.ContainsFunc(o.Safes, id.Equals) {
			label := o.Label
			if label == "" {
				label = o.ObjectId
			}
			used = append(used, fmt.Sprintf("%s %q", o.ObjectType, label))
		}
	}
	if len(used) > 0 {
		return happydns.CustomError{
			Err:    fmt.Errorf("this keystore still holds the secrets of %s: migrate them to another keystore first", strings.Join(used, ", ")),
			Status: http.StatusConflict,
		}
	}

	return keystoreError(u.manager.DeleteKeystore(ctx, owner, ref, instance))
}

// Migrate moves the secrets owner keeps in the keystore from into the
// keystore to. An object that cannot move, as one used in the background
// toward a keystore needing its owner, is reported and left where it is.
func (u *KeystoreUsecase) Migrate(ctx context.Context, owner happydns.Identifier, from, to string) ([]secret.ResealReport, error) {
	if from == to {
		return nil, happydns.ValidationError{Msg: "a keystore cannot be migrated into itself"}
	}
	fromId, err := u.manager.KeystoreSafe(ctx, owner, from)
	if errors.Is(err, happydns.ErrSafeNotFound) {
		// Not created yet: it holds nothing.
		return []secret.ResealReport{}, nil
	}
	if err != nil {
		return nil, keystoreError(err)
	}
	if err := u.manager.CheckMigrationTarget(ctx, owner, to); err != nil {
		return nil, keystoreError(err)
	}

	reports := []secret.ResealReport{}
	for _, t := range slices.Sorted(maps.Keys(u.holders)) {
		report, err := u.holders[t].MoveOwnerSecrets(ctx, owner, fromId, to)
		if err != nil {
			return reports, fmt.Errorf("%s: %w", t, err)
		}
		reports = append(reports, report)
	}
	return reports, nil
}

// SetDefault records ref as the keystore new objects of owner go to; empty,
// the policy applies again.
func (u *KeystoreUsecase) SetDefault(ctx context.Context, owner happydns.Identifier, ref string) error {
	return keystoreError(u.manager.SetDefaultKeystore(ctx, owner, ref))
}
