// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	checkerPkg "git.happydns.org/happyDomain/internal/dnschecker"
	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/model"
)

// fieldMetaCache caches the result of computeFieldMeta per CheckerDefinition.
// Checker definitions are immutable after init-time registration, so the cache
// never needs invalidation.
var fieldMetaCache sync.Map // *happydns.CheckerDefinition -> checkerFieldMeta

// isEmptyValue returns true if v is nil or an empty string.
func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok && s == "" {
		return true
	}
	return false
}

// identifiersEqual returns true when both identifiers are nil or point to the same value.
func identifiersEqual(a, b *happydns.Identifier) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equals(*b)
}

// getScopedOptions returns options stored exactly at the requested scope level,
// without merging parent scopes.
func (u *CheckerOptionsUsecase) getScopedOptions(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
) (happydns.CheckerOptions, error) {
	positionals, err := u.store.GetCheckerConfiguration(checkerName, userId, domainId, serviceId)
	if err != nil {
		return make(happydns.CheckerOptions), err
	}
	for _, p := range positionals {
		if identifiersEqual(p.UserId, userId) && identifiersEqual(p.DomainId, domainId) && identifiersEqual(p.ServiceId, serviceId) {
			// Return a copy: callers (MergeCheckerOptions, SetCheckerOption)
			// mutate the result before persisting, and the store may return a
			// shared reference to its in-memory state.
			out := make(happydns.CheckerOptions, len(p.Options))
			maps.Copy(out, p.Options)
			return out, nil
		}
	}
	return make(happydns.CheckerOptions), nil
}

// filterOptionsForScope drops keys that must not be persisted at the given
// scope: auto-fill keys (system-provided at runtime) and NoOverride keys
// declared at a broader scope. Returns the metadata used so callers can reuse
// it for further work without re-deriving it.
func (u *CheckerOptionsUsecase) filterOptionsForScope(
	checkerName string,
	scope happydns.CheckScopeType,
	opts happydns.CheckerOptions,
) (happydns.CheckerOptions, checkerFieldMeta) {
	var meta checkerFieldMeta
	if def := checkerPkg.FindChecker(checkerName); def != nil {
		meta = computeFieldMeta(def)
	}

	filtered := make(happydns.CheckerOptions, len(opts))
	for k, v := range opts {
		if meta.autoFillIds[k] != "" {
			continue
		}
		if defScope, ok := meta.noOverrideScopes[k]; ok && scope > defScope {
			continue
		}
		filtered[k] = v
	}
	return filtered, meta
}

// mergePositionalsRespectingNoOverride merges positionals from least to most
// specific scope. Keys flagged NoOverride keep their first-seen (least-specific)
// value and are not overridden by finer scopes.
//
// The input slice is sorted by scope here rather than trusting the caller (or
// the store) to return positionals in any particular order: the merge result
// only makes sense least-to-most-specific.
func mergePositionalsRespectingNoOverride(
	positionals []*happydns.CheckerOptionsPositional,
	noOverrideIds map[string]bool,
) happydns.CheckerOptions {
	ordered := make([]*happydns.CheckerOptionsPositional, len(positionals))
	copy(ordered, positionals)
	slices.SortStableFunc(ordered, func(a, b *happydns.CheckerOptionsPositional) int {
		sa := scopeFromIdentifiers(a.UserId, a.DomainId, a.ServiceId)
		sb := scopeFromIdentifiers(b.UserId, b.DomainId, b.ServiceId)
		return int(sa) - int(sb)
	})

	merged := make(happydns.CheckerOptions)
	for _, p := range ordered {
		for k, v := range p.Options {
			if noOverrideIds[k] {
				if _, exists := merged[k]; exists {
					continue
				}
			}
			merged[k] = v
		}
	}
	return merged
}

// CheckerOptionsUsecase handles the resolution and persistence of checker options.
type CheckerOptionsUsecase struct {
	store          CheckerOptionsStorage
	autoFillStore  CheckAutoFillStorage
	discoveryStore DiscoveryEntryStorage
	adminOptions   map[string]happydns.CheckerOptions
}

// NewCheckerOptionsUsecase creates a new CheckerOptionsUsecase.
func NewCheckerOptionsUsecase(store CheckerOptionsStorage, autoFillStore CheckAutoFillStorage) *CheckerOptionsUsecase {
	return &CheckerOptionsUsecase{store: store, autoFillStore: autoFillStore}
}

// WithDiscoveryEntryStore enables AutoFillDiscoveryEntries: options fields
// declaring that auto-fill will be populated with the entries stored for the
// target (see docs/checker-discovery.md). Passing nil (or not calling this)
// keeps AutoFillDiscoveryEntries fields unfilled.
func (u *CheckerOptionsUsecase) WithDiscoveryEntryStore(store DiscoveryEntryStorage) *CheckerOptionsUsecase {
	u.discoveryStore = store
	return u
}

// WithAdminOptions installs per-checker admin-scope option overrides sourced
// from CLI flags / env vars. They are applied with the highest priority in
// GetCheckerOptions (so the admin panel reflects effective values) and in
// BuildMergedCheckerOptionsWithAutoFill (so executions use them). Calling
// with nil or an empty map is a no-op.
func (u *CheckerOptionsUsecase) WithAdminOptions(opts map[string]happydns.CheckerOptions) *CheckerOptionsUsecase {
	u.adminOptions = opts
	return u
}

// GetCheckerOptionsPositional returns the raw positional options from all scope levels,
// ordered from least to most specific (admin < user < domain < service).
func (u *CheckerOptionsUsecase) GetCheckerOptionsPositional(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
) ([]*happydns.CheckerOptionsPositional, error) {
	return u.store.GetCheckerConfiguration(checkerName, userId, domainId, serviceId)
}

// GetAutoFillOptions resolves auto-fill values for a checker and target,
// returning only the auto-filled key/value pairs. Returns nil (not an empty
// map) when there is nothing to fill, so callers can use a simple len check.
func (u *CheckerOptionsUsecase) GetAutoFillOptions(
	checkerName string,
	target happydns.CheckTarget,
) (happydns.CheckerOptions, error) {
	def := checkerPkg.FindChecker(checkerName)
	if def == nil {
		return nil, nil
	}

	autoFillFields := computeFieldMeta(def).autoFillIds
	if len(autoFillFields) == 0 {
		return nil, nil
	}

	ctx, err := u.buildAutoFillContext(target)
	if err != nil {
		return nil, err
	}

	result := make(happydns.CheckerOptions, len(autoFillFields))
	for fieldId, autoFillKey := range autoFillFields {
		if val, ok := ctx[autoFillKey]; ok {
			result[fieldId] = val
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// GetCheckerOptions retrieves and merges options from all applicable levels
// (admin < user < domain < service), returning the merged result.
func (u *CheckerOptionsUsecase) GetCheckerOptions(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
) (happydns.CheckerOptions, error) {
	positionals, err := u.store.GetCheckerConfiguration(checkerName, userId, domainId, serviceId)
	if err != nil {
		return nil, err
	}
	merged, _ := u.mergeStoredOptions(checkerName, positionals)
	u.overlayCLIAdmin(checkerName, merged)
	return merged, nil
}

// mergeStoredOptions merges per-scope positionals (least to most specific,
// respecting NoOverride). The pre-computed field metadata is returned so
// callers that already need it (auto-fill resolution, NoOverride checks on
// runOpts) don't recompute it.
func (u *CheckerOptionsUsecase) mergeStoredOptions(
	checkerName string,
	positionals []*happydns.CheckerOptionsPositional,
) (happydns.CheckerOptions, checkerFieldMeta) {
	var meta checkerFieldMeta
	if def := checkerPkg.FindChecker(checkerName); def != nil {
		meta = computeFieldMeta(def)
	}
	return mergePositionalsRespectingNoOverride(positionals, meta.noOverrideIds), meta
}

// overlayCLIAdmin applies CLI/env admin overrides on top of dst, in place.
func (u *CheckerOptionsUsecase) overlayCLIAdmin(checkerName string, dst happydns.CheckerOptions) {
	if cliAdmin := u.adminOptions[checkerName]; len(cliAdmin) > 0 {
		maps.Copy(dst, cliAdmin)
	}
}

// SetCheckerOptions persists options at the given positional level (full replace).
// Keys with nil or empty-string values are excluded from the stored map.
// Auto-fill keys are also stripped since they are system-provided at runtime.
func (u *CheckerOptionsUsecase) SetCheckerOptions(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
	opts happydns.CheckerOptions,
) error {
	// Drop empties first so filterOptionsForScope doesn't have to handle them.
	nonEmpty := make(happydns.CheckerOptions, len(opts))
	for k, v := range opts {
		if !isEmptyValue(v) {
			nonEmpty[k] = v
		}
	}
	scope := scopeFromIdentifiers(userId, domainId, serviceId)
	filtered, _ := u.filterOptionsForScope(checkerName, scope, nonEmpty)
	return u.store.UpdateCheckerConfiguration(checkerName, userId, domainId, serviceId, filtered)
}

// MergeCheckerOptions computes the result of merging newOpts into the existing
// options at the given scope level WITHOUT persisting it. This allows callers to
// validate the merged result before committing it to storage.
// Keys with nil or empty-string values are removed from the merged map.
func (u *CheckerOptionsUsecase) MergeCheckerOptions(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
	newOpts happydns.CheckerOptions,
) (happydns.CheckerOptions, error) {
	existing, err := u.getScopedOptions(checkerName, userId, domainId, serviceId)
	if err != nil {
		return nil, err
	}

	scope := scopeFromIdentifiers(userId, domainId, serviceId)

	// Filter newOpts down to keys we are allowed to persist at this scope.
	// Pass only non-empties through the filter; empties are sentinels for
	// deletion and must reach the merge step regardless.
	nonEmpty := make(happydns.CheckerOptions, len(newOpts))
	for k, v := range newOpts {
		if !isEmptyValue(v) {
			nonEmpty[k] = v
		}
	}
	filteredNew, meta := u.filterOptionsForScope(checkerName, scope, nonEmpty)

	// Defense-in-depth: strip any auto-fill keys already in `existing` in case
	// older records leaked them in before this filter existed.
	for k := range meta.autoFillIds {
		delete(existing, k)
	}

	maps.Copy(existing, filteredNew)
	for k, v := range newOpts {
		if isEmptyValue(v) {
			delete(existing, k)
		}
	}
	return existing, nil
}

// AddCheckerOptions merges new options into existing ones at the given scope level
// and persists the result. Keys with nil or empty-string values are deleted from the
// scope rather than stored.
func (u *CheckerOptionsUsecase) AddCheckerOptions(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
	newOpts happydns.CheckerOptions,
) (happydns.CheckerOptions, error) {
	merged, err := u.MergeCheckerOptions(checkerName, userId, domainId, serviceId, newOpts)
	if err != nil {
		return nil, err
	}
	if err := u.store.UpdateCheckerConfiguration(checkerName, userId, domainId, serviceId, merged); err != nil {
		return nil, err
	}
	return merged, nil
}

// GetCheckerOption returns a single option value from the merged options.
func (u *CheckerOptionsUsecase) GetCheckerOption(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
	optName string,
) (any, error) {
	opts, err := u.GetCheckerOptions(checkerName, userId, domainId, serviceId)
	if err != nil {
		return nil, err
	}
	return opts[optName], nil
}

// scopeFromIdentifiers determines the CheckScopeType based on which identifiers are set.
func scopeFromIdentifiers(userId, domainId, serviceId *happydns.Identifier) happydns.CheckScopeType {
	if serviceId != nil {
		return happydns.CheckScopeService
	}
	if domainId != nil {
		return happydns.CheckScopeDomain
	}
	if userId != nil {
		return happydns.CheckScopeUser
	}
	return happydns.CheckScopeAdmin
}

// appendBroaderScopeFields copies fields from a broader scope into dst as
// keys accepted at a finer scope. The Required flag is stripped because the
// value may already be set at the broader scope and need not be repeated.
// If skipNoOverride is true, fields marked NoOverride are dropped entirely.
func appendBroaderScopeFields(dst, src []happydns.CheckerOptionDocumentation, skipNoOverride bool) []happydns.CheckerOptionDocumentation {
	for _, f := range src {
		if skipNoOverride && f.NoOverride {
			continue
		}
		f.Required = false
		dst = append(dst, f)
	}
	return dst
}

// collectFieldsForScope returns the fields from a CheckerOptionsDocumentation
// that are valid at the given scope level. A more specific scope also accepts
// any broader-scope field that is not marked NoOverride, since values set at
// a broader scope can be overridden here. RunOpts are never included for
// persisted scopes.
func collectFieldsForScope(doc happydns.CheckerOptionsDocumentation, scope happydns.CheckScopeType) []happydns.CheckerOptionDocumentation {
	var fields []happydns.CheckerOptionDocumentation
	switch scope {
	case happydns.CheckScopeAdmin:
		fields = append(fields, doc.AdminOpts...)
	case happydns.CheckScopeUser:
		fields = appendBroaderScopeFields(fields, doc.AdminOpts, true)
		fields = append(fields, doc.UserOpts...)
	case happydns.CheckScopeDomain, happydns.CheckScopeZone:
		fields = appendBroaderScopeFields(fields, doc.AdminOpts, true)
		fields = appendBroaderScopeFields(fields, doc.UserOpts, true)
		fields = append(fields, doc.DomainOpts...)
	case happydns.CheckScopeService:
		fields = appendBroaderScopeFields(fields, doc.AdminOpts, true)
		fields = appendBroaderScopeFields(fields, doc.UserOpts, true)
		fields = appendBroaderScopeFields(fields, doc.DomainOpts, true)
		fields = append(fields, doc.ServiceOpts...)
	}
	return fields
}

// collectValidatableFields gathers the option fields that should be validated
// for the given scope, including fields contributed by rules. When withRunOpts
// is true (trigger time), all persisted-scope fields are accepted as keys
// (with Required stripped) so that values already merged from a persisted
// scope aren't rejected as unknown.
func collectValidatableFields(
	def *happydns.CheckerDefinition,
	scope happydns.CheckScopeType,
	withRunOpts bool,
) []happydns.CheckerOptionDocumentation {
	collectFromDoc := func(dst []happydns.CheckerOptionDocumentation, doc happydns.CheckerOptionsDocumentation) []happydns.CheckerOptionDocumentation {
		if !withRunOpts {
			return append(dst, collectFieldsForScope(doc, scope)...)
		}
		dst = appendBroaderScopeFields(dst, doc.AdminOpts, false)
		dst = appendBroaderScopeFields(dst, doc.UserOpts, false)
		dst = appendBroaderScopeFields(dst, doc.DomainOpts, false)
		dst = appendBroaderScopeFields(dst, doc.ServiceOpts, false)
		return append(dst, doc.RunOpts...)
	}

	var fields []happydns.CheckerOptionDocumentation
	fields = collectFromDoc(fields, def.Options)
	for _, rule := range def.Rules {
		if rwo, ok := rule.(happydns.CheckRuleWithOptions); ok {
			fields = collectFromDoc(fields, rwo.Options())
		}
	}
	return fields
}

// validateSingleOption validates value against the field schema declared for
// optName. Unknown keys (not declared anywhere in the checker) are rejected.
func validateSingleOption(def *happydns.CheckerDefinition, optName string, value any) error {
	field, ok := computeFieldMeta(def).fields[optName]
	if !ok {
		return fmt.Errorf("option %q is not declared by checker %q", optName, def.Name)
	}
	return forms.ValidateMapValues(
		happydns.CheckerOptions{optName: value},
		[]happydns.Field{happydns.FieldFromCheckerOption(field)},
	)
}

// ValidateOptions validates checker options against the checker's field definitions
// for the given scope level, and any OptionsValidator interface implemented by rules.
// When withRunOpts is true, RunOpts fields are also included so that required run-time
// options are enforced (used at trigger time). For persisted scopes, pass false.
func (u *CheckerOptionsUsecase) ValidateOptions(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
	opts happydns.CheckerOptions,
	withRunOpts bool,
) error {
	def := checkerPkg.FindChecker(checkerName)
	if def == nil {
		return fmt.Errorf("checker %q not found", checkerName)
	}

	scope := scopeFromIdentifiers(userId, domainId, serviceId)
	allFields := collectValidatableFields(def, scope, withRunOpts)

	// Strip auto-fill fields: they are system-provided at runtime and should
	// not be validated against user input.
	autoFillIds := computeFieldMeta(def).autoFillIds
	asFields := make([]happydns.Field, 0, len(allFields))
	for _, f := range allFields {
		if _, isAutoFill := autoFillIds[f.Id]; isAutoFill {
			continue
		}
		// CheckerOptionDocumentation is structurally identical to happydns.Field;
		// forms.ValidateMapValues operates on the latter.
		asFields = append(asFields, happydns.FieldFromCheckerOption(f))
	}

	if len(asFields) > 0 {
		if err := forms.ValidateMapValues(opts, asFields); err != nil {
			return err
		}
	}

	// Rule-level semantic validation (beyond field shape).
	for _, rule := range def.Rules {
		if v, ok := rule.(happydns.OptionsValidator); ok {
			if err := v.ValidateOptions(opts); err != nil {
				return err
			}
		}
	}

	return nil
}

// SetCheckerOption sets a single option value at the given scope level.
// If value is nil or empty string, the key is deleted from the scope.
func (u *CheckerOptionsUsecase) SetCheckerOption(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
	optName string,
	value any,
) error {
	if def := checkerPkg.FindChecker(checkerName); def != nil {
		meta := computeFieldMeta(def)
		// Auto-fill keys are system-provided at runtime; never persist them.
		if meta.autoFillIds[optName] != "" {
			return fmt.Errorf("option %q is auto-filled and cannot be set", optName)
		}
		// Defense-in-depth: reject NoOverride fields at scopes below their definition.
		if defScope, ok := meta.noOverrideScopes[optName]; ok {
			currentScope := scopeFromIdentifiers(userId, domainId, serviceId)
			if currentScope > defScope {
				return fmt.Errorf("option %q cannot be overridden at this scope level", optName)
			}
		}
		// Validate the value against its field schema. Deletions (empty value)
		// skip shape validation: the key is being removed, not stored.
		if !isEmptyValue(value) {
			if err := validateSingleOption(def, optName, value); err != nil {
				return err
			}
		}
	}

	existing, err := u.getScopedOptions(checkerName, userId, domainId, serviceId)
	if err != nil {
		return err
	}
	if isEmptyValue(value) {
		delete(existing, optName)
	} else {
		existing[optName] = value
	}
	return u.store.UpdateCheckerConfiguration(checkerName, userId, domainId, serviceId, existing)
}

// checkerFieldMeta holds pre-computed field metadata for a checker definition,
// avoiding repeated scans of the same option groups and rules.
type checkerFieldMeta struct {
	autoFillIds      map[string]string
	noOverrideIds    map[string]bool
	noOverrideScopes map[string]happydns.CheckScopeType
	// fields indexes every declared option field by Id (first declaration wins,
	// matching the scope precedence Admin→User→Domain→Service→Run, then rules).
	fields map[string]happydns.CheckerOptionDocumentation
}

// computeFieldMeta returns cached field metadata for a checker definition.
// The result is computed once per definition and cached for the process lifetime.
func computeFieldMeta(def *happydns.CheckerDefinition) checkerFieldMeta {
	if cached, ok := fieldMetaCache.Load(def); ok {
		return cached.(checkerFieldMeta)
	}
	meta := buildFieldMeta(def)
	fieldMetaCache.Store(def, meta)
	return meta
}

// buildFieldMeta scans all option groups and rules of a checker definition
// and returns the consolidated field metadata.
func buildFieldMeta(def *happydns.CheckerDefinition) checkerFieldMeta {
	meta := checkerFieldMeta{
		autoFillIds:      make(map[string]string),
		noOverrideIds:    make(map[string]bool),
		noOverrideScopes: make(map[string]happydns.CheckScopeType),
		fields:           make(map[string]happydns.CheckerOptionDocumentation),
	}

	scanDoc := func(doc happydns.CheckerOptionsDocumentation) {
		// AutoFill is meaningful at every scope including RunOpts: a per-run
		// field can legitimately be system-populated. The field index also
		// covers all groups; first declaration wins to match the lookup order
		// Admin→User→Domain→Service→Run, then rules.
		allGroups := [][]happydns.CheckerOptionDocumentation{
			doc.AdminOpts, doc.UserOpts, doc.DomainOpts, doc.ServiceOpts, doc.RunOpts,
		}
		for _, fields := range allGroups {
			for _, f := range fields {
				if f.AutoFill != "" {
					meta.autoFillIds[f.Id] = f.AutoFill
				}
				if _, exists := meta.fields[f.Id]; !exists {
					meta.fields[f.Id] = f
				}
			}
		}

		// NoOverride is a precedence rule between persisted scopes; it has no
		// meaning for RunOpts (never persisted, supplied per-execution).
		type scopedGroup struct {
			fields []happydns.CheckerOptionDocumentation
			scope  happydns.CheckScopeType
		}
		persistedGroups := []scopedGroup{
			{doc.AdminOpts, happydns.CheckScopeAdmin},
			{doc.UserOpts, happydns.CheckScopeUser},
			{doc.DomainOpts, happydns.CheckScopeDomain},
			{doc.ServiceOpts, happydns.CheckScopeService},
		}
		for _, g := range persistedGroups {
			for _, f := range g.fields {
				if f.NoOverride {
					meta.noOverrideIds[f.Id] = true
					meta.noOverrideScopes[f.Id] = g.scope
				}
			}
		}
	}

	scanDoc(def.Options)
	for _, rule := range def.Rules {
		if rwo, ok := rule.(happydns.CheckRuleWithOptions); ok {
			scanDoc(rwo.Options())
		}
	}
	return meta
}

// buildAutoFillContext loads domain/zone data from storage and builds a map
// of auto-fill key to resolved value.
func (u *CheckerOptionsUsecase) buildAutoFillContext(
	target happydns.CheckTarget,
) (map[string]any, error) {
	ctx := make(map[string]any)
	if u.autoFillStore == nil {
		return ctx, nil
	}

	domainId := happydns.TargetIdentifier(target.DomainId)
	if domainId == nil {
		return ctx, nil
	}

	domain, err := u.autoFillStore.GetDomain(*domainId)
	if err != nil {
		return ctx, fmt.Errorf("loading domain for auto-fill: %w", err)
	}

	ctx[happydns.AutoFillDomainName] = domain.DomainName

	// Load the WIP zone ([0]) for auto-fill context, so the user can
	// configure checkers for services they are currently working on.
	if len(domain.ZoneHistory) == 0 {
		return ctx, nil
	}

	zone, err := u.autoFillStore.GetZone(domain.ZoneHistory[0])
	if err != nil {
		return ctx, fmt.Errorf("loading zone for auto-fill: %w", err)
	}
	ctx[happydns.AutoFillZone] = zone

	// Resolve service if target has a ServiceId.
	// Search WIP first, then latest published, then older history.
	if serviceId := happydns.TargetIdentifier(target.ServiceId); serviceId != nil {
		for i := 0; i < len(domain.ZoneHistory); i++ {
			z := zone
			if i > 0 {
				z, err = u.autoFillStore.GetZone(domain.ZoneHistory[i])
				if err != nil {
					if errors.Is(err, happydns.ErrZoneNotFound) {
						continue
					}
					return ctx, fmt.Errorf("loading zone for auto-fill: %w", err)
				}
			}
			for subdomain, services := range z.Services {
				for _, svc := range services {
					if svc.Id.Equals(*serviceId) {
						ctx[happydns.AutoFillSubdomain] = string(subdomain)
						ctx[happydns.AutoFillServiceType] = svc.Type
						ctx[happydns.AutoFillService] = svc
						return ctx, nil
					}
				}
			}
		}
	}

	return ctx, nil
}

// BuildMergedCheckerOptionsWithAutoFill produces the final option map fed to a
// checker execution. Precedence, from lowest to highest:
//
//	stored(admin → user → domain → service) → runOpts → auto-fill → CLI admin
//
// NoOverride fields are honored across stored scopes and reject runOpts
// entirely (they may only be set at their declaration scope or via CLI admin).
//
// The second return value is the set of DiscoveryEntry records injected into
// AutoFillDiscoveryEntries fields (if any): exposed so the engine can
// persist the consumer→entry lineage after the run completes. It is nil
// when no such field was auto-filled.
func (u *CheckerOptionsUsecase) BuildMergedCheckerOptionsWithAutoFill(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
	runOpts happydns.CheckerOptions,
) (happydns.CheckerOptions, []*happydns.StoredDiscoveryEntry, error) {
	positionals, err := u.store.GetCheckerConfiguration(checkerName, userId, domainId, serviceId)
	if err != nil {
		return nil, nil, err
	}

	storedOpts, meta := u.mergeStoredOptions(checkerName, positionals)

	// Apply runtime overrides on top. NoOverride fields are owned by their
	// declaration scope (admin/user/domain/service) and cannot be supplied via
	// the trigger payload, whether or not a stored value already exists.
	merged := make(happydns.CheckerOptions, len(storedOpts)+len(runOpts))
	maps.Copy(merged, storedOpts)
	for k, v := range runOpts {
		if meta.noOverrideIds[k] {
			continue
		}
		merged[k] = v
	}

	var injectedEntries []*happydns.StoredDiscoveryEntry

	// Resolve auto-fill values (always win). meta is zero when def is nil,
	// so the len check alone is sufficient.
	if len(meta.autoFillIds) > 0 {
		target := happydns.CheckTarget{
			UserId:    happydns.FormatIdentifier(userId),
			DomainId:  happydns.FormatIdentifier(domainId),
			ServiceId: happydns.FormatIdentifier(serviceId),
		}
		ctx, err := u.buildAutoFillContext(target)
		if err != nil {
			return nil, nil, err
		}

		// AutoFillDiscoveryEntries is resolved from a separate storage surface
		// (the discovery entry index), loaded lazily on first encounter.
		var discoveryEntries []*happydns.StoredDiscoveryEntry
		var discoveryLoaded bool

		for fieldId, autoFillKey := range meta.autoFillIds {
			if autoFillKey == happydns.AutoFillDiscoveryEntries {
				if !discoveryLoaded {
					discoveryLoaded = true
					if u.discoveryStore != nil {
						discoveryEntries, err = u.discoveryStore.ListDiscoveryEntriesByTarget(target)
						if err != nil {
							return nil, nil, fmt.Errorf("loading discovery entries: %w", err)
						}
						if len(discoveryEntries) > 0 {
							injectedEntries = discoveryEntries
						}
					}
				}
				merged[fieldId] = sdkEntries(discoveryEntries)
				continue
			}
			if val, ok := ctx[autoFillKey]; ok {
				merged[fieldId] = val
			}
		}
	}

	// CLI admin opts win over everything, including auto-fill.
	u.overlayCLIAdmin(checkerName, merged)

	return merged, injectedEntries, nil
}

// sdkEntries converts host-side StoredDiscoveryEntry values to the opaque
// SDK-level DiscoveryEntry form that is passed to consumer checkers. The
// producer/target namespacing is not exposed to the consumer: it would be
// meaningless in that contract.
func sdkEntries(stored []*happydns.StoredDiscoveryEntry) []happydns.DiscoveryEntry {
	out := make([]happydns.DiscoveryEntry, 0, len(stored))
	for _, e := range stored {
		out = append(out, happydns.DiscoveryEntry{
			Type:    e.Type,
			Ref:     e.Ref,
			Payload: e.Payload,
		})
	}
	return out
}
