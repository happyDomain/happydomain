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
	"errors"
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

// secretIdsOf lists the options the documentation of the checker sets
// Secret. It says which values to seal; to tell which stored values are
// secrets, use secretKeys.
func secretIdsOf(checkerName string) map[string]bool {
	def := checkerPkg.FindChecker(checkerName)
	if def == nil {
		return nil
	}
	return computeFieldMeta(def).secretIds
}

// isSealedValue reports whether v is a sealed value.
func isSealedValue(v any) bool {
	s, ok := v.(string)
	return ok && secret.IsSealed(s)
}

// secretKeys lists the options of opts holding a secret: those documented
// Secret, and any holding a sealed value. A sealed value stays a secret when
// its checker is gone or no longer documents it so.
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

// eachSecretSet calls fn with each secret option of opts (see secretKeys) set
// to a non-empty string, and the context it is bound to.
func eachSecretSet(checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions, fn func(key, value string, sc secret.SecretContext) error) error {
	for k := range secretKeys(checkerName, opts) {
		s, ok := opts[k].(string)
		if !ok || s == "" {
			continue
		}
		if err := fn(k, s, OptionsSecretContext(checkerName, userId, domainId, serviceId, k)); err != nil {
			return fmt.Errorf("option %q: %w", k, err)
		}
	}
	return nil
}

// secretString returns v, the value of the secret option key, as the string
// every secret has to be.
func secretString(key string, v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", happydns.ValidationError{Msg: fmt.Sprintf("option %q is secret and must be a string", key)}
	}
	return s, nil
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
	if isSealedValue(value) {
		return happydns.ValidationError{Msg: fmt.Sprintf("option %q: %s", key, secret.ErrSealedFromClient)}
	}
	if isSecret && !isEmptyValue(value) {
		if _, err := secretString(key, value); err != nil {
			return err
		}
	}
	return nil
}

// hasPlaceholder reports whether a value of opts is the placeholder.
func hasPlaceholder(opts happydns.CheckerOptions) bool {
	for _, v := range opts {
		if v == happydns.RedactedSecret {
			return true
		}
	}
	return false
}

// resolveEchoes returns a copy of opts where a value the client echoes back
// as the placeholder, which it only ever received in place of a secret,
// takes the value stored in existing, or is dropped when nothing is stored.
func resolveEchoes(existing, opts happydns.CheckerOptions) happydns.CheckerOptions {
	out := make(happydns.CheckerOptions, len(opts))
	for k, v := range opts {
		if v != happydns.RedactedSecret {
			out[k] = v
		} else if stored, ok := existing[k]; ok {
			out[k] = stored
		}
	}
	return out
}

// sealOptions seals, in place, the secret options of one scope before they
// are stored. Values already sealed, read from storage, are kept.
func (u *CheckerOptionsUsecase) sealOptions(checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error {
	return sealOptionsWith(u.secrets, checkerName, userId, domainId, serviceId, opts)
}

// RestoreOptions returns opts, the options of one scope read from a backup,
// as they have to be stored. The placeholders a user export carries in place
// of the values it withholds stand for no value, and are dropped. Secret
// options in clear are sealed under the current policy. A sealed value is
// kept only if it opens here.
func RestoreOptions(secrets *secret.Manager, checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions) (happydns.CheckerOptions, error) {
	out := resolveEchoes(nil, opts)

	if err := openOptionsWith(secrets.NewValueOpener(), checkerName, userId, domainId, serviceId, maps.Clone(out)); err != nil {
		return nil, err
	}
	if err := sealOptionsWith(secrets, checkerName, userId, domainId, serviceId, out); err != nil {
		return nil, err
	}
	return out, nil
}

func sealOptionsWith(secrets *secret.Manager, checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error {
	secretIds := secretIdsOf(checkerName)
	for k, v := range opts {
		if v == happydns.RedactedSecret {
			return fmt.Errorf("option %q: %w", k, secret.ErrRedactedSecret)
		}
		if !secretIds[k] || isEmptyValue(v) {
			continue
		}
		s, err := secretString(k, v)
		if err != nil {
			return err
		}

		sealed, err := secrets.SealValue(context.Background(), OptionsSecretContext(checkerName, userId, domainId, serviceId, k), s)
		if err != nil {
			return fmt.Errorf("unable to seal option %q: %w", k, err)
		}
		opts[k] = sealed
	}
	return nil
}

// openOptions opens, in place, the secret options of one scope, as stored.
func (u *CheckerOptionsUsecase) openOptions(checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error {
	return openOptionsWith(u.secrets.NewValueOpener(), checkerName, userId, domainId, serviceId, opts)
}

func openOptionsWith(opener *secret.ValueOpener, checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error {
	// Writing opts is safe: secretKeys returns a map of its own.
	return eachSecretSet(checkerName, userId, domainId, serviceId, opts, func(k, s string, sc secret.SecretContext) error {
		clear, err := opener.Open(context.Background(), sc, s)
		if err != nil {
			return err
		}
		opts[k] = clear
		return nil
	})
}

// openPositionals returns copies of positionals with their secret options
// opened, each with the context of its own scope.
func (u *CheckerOptionsUsecase) openPositionals(positionals []*happydns.CheckerOptionsPositional) ([]*happydns.CheckerOptionsPositional, error) {
	// One opener for every scope: each safe is read once.
	opener := u.secrets.NewValueOpener()
	out := make([]*happydns.CheckerOptionsPositional, 0, len(positionals))
	for _, p := range positionals {
		if len(secretKeys(p.CheckName, p.Options)) == 0 {
			out = append(out, p)
			continue
		}

		cp := *p
		cp.Options = maps.Clone(p.Options)
		if err := openOptionsWith(opener, p.CheckName, p.UserId, p.DomainId, p.ServiceId, cp.Options); err != nil {
			return nil, err
		}
		out = append(out, &cp)
	}
	return out, nil
}

// optionsForValidation returns opts as validation has to see them: the
// placeholder a client echoes back stands for the value stored at this
// scope, opened, or at run time for whatever is stored. What a client may not
// send, such as a sealed value, is refused.
func (u *CheckerOptionsUsecase) optionsForValidation(checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions, withRunOpts bool) (happydns.CheckerOptions, error) {
	if err := checkIncomingOptions(checkerName, opts); err != nil {
		return nil, err
	}
	if !hasPlaceholder(opts) {
		return opts, nil
	}
	if withRunOpts {
		// Dropped: the run uses what is stored.
		return resolveEchoes(nil, opts), nil
	}

	existing, err := u.getScopedOptions(checkerName, userId, domainId, serviceId)
	if err != nil {
		return nil, err
	}
	if err := u.openOptions(checkerName, userId, domainId, serviceId, existing); err != nil {
		return nil, err
	}
	return resolveEchoes(existing, opts), nil
}

// RedactCheckerOptions returns a copy of opts where every secret option set
// is replaced by happydns.RedactedSecret, for the API.
func (u *CheckerOptionsUsecase) RedactCheckerOptions(checkerName string, opts happydns.CheckerOptions) happydns.CheckerOptions {
	return RedactOptions(checkerName, opts)
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

// withheld reports whether the value of the option key leaves happyDomain as
// happydns.RedactedSecret rather than as is. A value set is withheld when it
// is a secret, documented so or sealed, and when the documentation does not
// describe the option at all: nothing then tells it is not a secret stored in
// clear under the plaintext policy. That is the case of every option of a
// checker this build does not load (meta nil), and of an option renamed or
// dropped since it was stored.
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
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
) (happydns.CheckerOptions, error) {
	positionals, err := u.store.GetCheckerConfiguration(checkerName, userId, domainId, serviceId)
	if err != nil {
		return nil, err
	}
	positionals, err = u.openPositionals(positionals)
	if err != nil {
		return nil, err
	}
	merged, _ := u.mergeStoredOptions(checkerName, positionals)
	u.overlayCLIAdmin(checkerName, merged)
	return merged, nil
}

// CheckerOptionsSecrets lets the administrator follow and migrate how the
// secret checker options are stored.
type CheckerOptionsSecrets struct {
	store   CheckerOptionsStorage
	secrets *secret.Manager
}

func NewCheckerOptionsSecrets(store CheckerOptionsStorage, secrets *secret.Manager) *CheckerOptionsSecrets {
	return &CheckerOptionsSecrets{store: store, secrets: secrets}
}

// scopeName names a scope of options in the reports.
func scopeName(p *happydns.CheckerOptionsPositional) string {
	return "options " + optionsObjectId(p.CheckName, p.UserId, p.DomainId, p.ServiceId)
}

// InspectSecrets tells how the secret options of every scope are stored.
func (cs *CheckerOptionsSecrets) InspectSecrets(ctx context.Context) (secret.Counts, error) {
	iter, err := cs.store.ListAllCheckerConfigurations()
	if err != nil {
		return secret.Counts{}, err
	}

	return secret.InspectAll(iter, scopeName, func(p *happydns.CheckerOptionsPositional, c *secret.Counts) error {
		// Past a value that fails, the others are still looked at, so that
		// every value sealed in a safe that cannot be read is counted.
		var errs error
		err := eachSecretSet(p.CheckName, p.UserId, p.DomainId, p.ServiceId, p.Options, func(k, s string, sc secret.SecretContext) error {
			if err := cs.secrets.InspectValue(ctx, sc, s, c); err != nil {
				errs = errors.Join(errs, fmt.Errorf("option %q: %w", k, err))
			}
			return nil
		})
		return errors.Join(err, errs)
	})
}

// ResealSecrets stores the secret options of every scope the way the current
// policy stores new secrets. A scope that fails is reported and skipped; run
// it again to resume.
func (cs *CheckerOptionsSecrets) ResealSecrets(ctx context.Context) (secret.ResealReport, error) {
	iter, err := cs.store.ListAllCheckerConfigurations()
	if err != nil {
		return secret.ResealReport{ObjectType: OptionsSecretObjectType}, err
	}

	return secret.ResealAll(OptionsSecretObjectType, iter, scopeName, func(p *happydns.CheckerOptionsPositional) (bool, error) {
		if len(secretKeys(p.CheckName, p.Options)) == 0 {
			return false, nil
		}
		return cs.resealScope(ctx, p.CheckName, p.UserId, p.DomainId, p.ServiceId)
	})
}

// resealScope reseals the options of one scope as stored now. The write is
// conditional: whatever another writer did in between wins, and the scope is
// left for the next run.
func (cs *CheckerOptionsSecrets) resealScope(ctx context.Context, checkerName string, userId, domainId, serviceId *happydns.Identifier) (bool, error) {
	changed := false
	err := cs.store.ReplaceCheckerConfiguration(checkerName, userId, domainId, serviceId, func(opts happydns.CheckerOptions) (happydns.CheckerOptions, error) {
		out := maps.Clone(opts)
		c := false
		err := eachSecretSet(checkerName, userId, domainId, serviceId, opts, func(k, s string, sc secret.SecretContext) error {
			v, ch, err := cs.secrets.ResealValue(ctx, sc, s)
			if err != nil {
				return err
			}
			if ch {
				out[k] = v
				c = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if !c {
			return nil, nil
		}
		changed = true
		return out, nil
	})
	if errors.Is(err, happydns.ErrNotFound) {
		// Deleted since it was listed.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return changed, nil
}
