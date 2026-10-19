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

package checker

import (
	"context"
	"fmt"
	"maps"

	checkerPkg "git.happydns.org/happyDomain/internal/dnschecker"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// OptionsSecretObjectType names checker options in the context their
// secrets are bound to. They are used by scheduled runs, in the background:
// their secrets must stay in safes that open without the user.
const OptionsSecretObjectType = "checker-options"

// WithSecrets installs the manager sealing the options whose documentation
// sets Secret before they are stored, and opening them for runs. Without it,
// storing or using a secret option fails.
func (u *CheckerOptionsUsecase) WithSecrets(secrets *secret.Manager) *CheckerOptionsUsecase {
	u.secrets = secrets
	return u
}

func optionsObjectId(checkerName string, userId, domainId, serviceId *happydns.Identifier) string {
	return fmt.Sprintf("%s|%s|%s|%s", checkerName, happydns.FormatIdentifier(userId), happydns.FormatIdentifier(domainId), happydns.FormatIdentifier(serviceId))
}

// OptionsSecretContext returns the context the secret options of one scope
// are bound to. Options of the admin scope belong to the instance.
func OptionsSecretContext(checkerName string, userId, domainId, serviceId *happydns.Identifier, key string) secret.SecretContext {
	owner := secret.InstanceOwner()
	if userId != nil {
		owner = *userId
	}
	return secret.SecretContext{
		Owner:      owner,
		ObjectType: OptionsSecretObjectType,
		ObjectId:   optionsObjectId(checkerName, userId, domainId, serviceId),
		Field:      key,
	}
}

func secretIdsOf(checkerName string) map[string]bool {
	def := checkerPkg.FindChecker(checkerName)
	if def == nil {
		return nil
	}
	return computeFieldMeta(def).secretIds
}

func isSealedValue(v any) bool {
	s, ok := v.(string)
	return ok && secret.IsSealed(s)
}

// secretKeys lists the options of opts holding a secret: those documented
// Secret, and any holding a sealed value. A sealed value stays a secret when
// its checker is gone or no longer documents it so: it is opened, counted
// and resealed all the same.
func secretKeys(checkerName string, opts happydns.CheckerOptions) map[string]bool {
	keys := maps.Clone(secretIdsOf(checkerName))
	for k, v := range opts {
		if isSealedValue(v) {
			if keys == nil {
				keys = map[string]bool{}
			}
			keys[k] = true
		}
	}
	return keys
}

// fieldMetaOf returns the field metadata of the checker, or nil for a checker
// this build does not load.
func fieldMetaOf(checkerName string) *checkerFieldMeta {
	def := checkerPkg.FindChecker(checkerName)
	if def == nil {
		return nil
	}
	meta := computeFieldMeta(def)
	return &meta
}

// withheld reports whether the value of key leaves happyDomain as
// happydns.RedactedSecret: a secret, documented or sealed, and any option the
// documentation does not describe (meta nil for an unloaded checker), as it
// may be a secret stored in clear under the plaintext policy.
func withheld(meta *checkerFieldMeta, key string, value any) bool {
	if isEmptyValue(value) {
		return false
	}
	if meta == nil || isSealedValue(value) {
		return true
	}
	if _, documented := meta.fields[key]; !documented {
		return true
	}
	return meta.secretIds[key]
}

// RedactOptions returns a copy of opts where every option withheld is
// replaced by happydns.RedactedSecret, for whatever leaves happyDomain: the
// API, the export of an account.
func RedactOptions(checkerName string, opts happydns.CheckerOptions) happydns.CheckerOptions {
	if opts == nil {
		return nil
	}

	meta := fieldMetaOf(checkerName)
	out := make(happydns.CheckerOptions, len(opts))
	for k, v := range opts {
		if withheld(meta, k, v) {
			out[k] = happydns.RedactedSecret
		} else {
			out[k] = v
		}
	}
	return out
}

// checkIncomingOptions refuses what a client sent when a value is sealed,
// whatever the option: a client has no reason to send one, and accepting it
// would let a user paste a value taken from elsewhere. It also refuses a
// secret option that is not a string.
func checkIncomingOptions(checkerName string, opts happydns.CheckerOptions) error {
	secretIds := secretIdsOf(checkerName)
	for k, v := range opts {
		if err := checkIncomingOption(k, v, secretIds[k]); err != nil {
			return err
		}
	}
	return nil
}

func checkIncomingOption(key string, value any, isSecret bool) error {
	if isEmptyValue(value) {
		return nil
	}
	if isSealedValue(value) {
		return happydns.ValidationError{Msg: fmt.Sprintf("option %q: %s", key, secret.ErrSealedFromClient)}
	}
	if _, ok := value.(string); isSecret && !ok {
		return happydns.ValidationError{Msg: fmt.Sprintf("option %q is secret and must be a string", key)}
	}
	return nil
}

// sealOptions seals, in place, the secret options of one scope before they
// are stored. Values already sealed, read from storage, are kept.
func (u *CheckerOptionsUsecase) sealOptions(ctx context.Context, checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error {
	for k := range secretKeys(checkerName, opts) {
		v, present := opts[k]
		if !present || isEmptyValue(v) {
			continue
		}
		s, ok := v.(string)
		if !ok {
			return happydns.ValidationError{Msg: fmt.Sprintf("option %q is secret and must be a string", k)}
		}
		if s == happydns.RedactedSecret {
			return fmt.Errorf("option %q: %w", k, secret.ErrRedactedSecret)
		}

		sealed, err := u.secrets.SealValue(ctx, OptionsSecretContext(checkerName, userId, domainId, serviceId, k), s)
		if err != nil {
			return fmt.Errorf("unable to seal option %q: %w", k, err)
		}
		opts[k] = sealed
	}
	return nil
}

// openPositionals returns copies of positionals with their secret options
// opened, each with the context of its own scope.
func (u *CheckerOptionsUsecase) openPositionals(ctx context.Context, positionals []*happydns.CheckerOptionsPositional) ([]*happydns.CheckerOptionsPositional, error) {
	out := make([]*happydns.CheckerOptionsPositional, 0, len(positionals))
	for _, p := range positionals {
		secretIds := secretKeys(p.CheckName, p.Options)
		if len(secretIds) == 0 {
			out = append(out, p)
			continue
		}

		cp := *p
		cp.Options = make(happydns.CheckerOptions, len(p.Options))
		maps.Copy(cp.Options, p.Options)

		for k := range secretIds {
			s, ok := cp.Options[k].(string)
			if !ok || s == "" {
				continue
			}
			clear, err := u.secrets.OpenValue(ctx, OptionsSecretContext(p.CheckName, p.UserId, p.DomainId, p.ServiceId, k), s)
			if err != nil {
				return nil, fmt.Errorf("unable to open option %q: %w", k, err)
			}
			cp.Options[k] = clear
		}
		out = append(out, &cp)
	}
	return out, nil
}

// RedactCheckerOptions is RedactOptions, for the API.
func (u *CheckerOptionsUsecase) RedactCheckerOptions(checkerName string, opts happydns.CheckerOptions) happydns.CheckerOptions {
	return RedactOptions(checkerName, opts)
}

// RedactCheckerOptionValue returns value, or happydns.RedactedSecret when
// RedactOptions would withhold it.
func (u *CheckerOptionsUsecase) RedactCheckerOptionValue(checkerName, optName string, value any) any {
	if withheld(fieldMetaOf(checkerName), optName, value) {
		return happydns.RedactedSecret
	}
	return value
}

// GetCheckerOptionsForUse is GetCheckerOptions with secret options opened,
// for code handing them to a checker, such as prechecks.
func (u *CheckerOptionsUsecase) GetCheckerOptionsForUse(
	ctx context.Context,
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
) (happydns.CheckerOptions, error) {
	positionals, err := u.store.GetCheckerConfiguration(checkerName, userId, domainId, serviceId)
	if err != nil {
		return nil, err
	}
	positionals, err = u.openPositionals(ctx, positionals)
	if err != nil {
		return nil, err
	}
	merged, _ := u.mergeStoredOptions(checkerName, positionals)
	u.overlayCLIAdmin(checkerName, merged)
	return merged, nil
}
