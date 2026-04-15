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
	"fmt"
	"maps"
	"sync"

	checkerPkg "git.happydns.org/happyDomain/internal/checker"
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
			if p.Options != nil {
				return p.Options, nil
			}
			return make(happydns.CheckerOptions), nil
		}
	}
	return make(happydns.CheckerOptions), nil
}

// CheckerOptionsUsecase handles the resolution and persistence of checker options.
type CheckerOptionsUsecase struct {
	store         CheckerOptionsStorage
	autoFillStore CheckAutoFillStorage
}

// NewCheckerOptionsUsecase creates a new CheckerOptionsUsecase.
func NewCheckerOptionsUsecase(store CheckerOptionsStorage, autoFillStore CheckAutoFillStorage) *CheckerOptionsUsecase {
	return &CheckerOptionsUsecase{store: store, autoFillStore: autoFillStore}
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
// returning only the auto-filled key/value pairs.
func (u *CheckerOptionsUsecase) GetAutoFillOptions(
	checkerName string,
	target happydns.CheckTarget,
) (happydns.CheckerOptions, error) {
	result, err := u.resolveAutoFill(checkerName, target)
	if err != nil {
		return nil, err
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

	// Determine which fields are NoOverride.
	var noOverrideIds map[string]bool
	if def := checkerPkg.FindChecker(checkerName); def != nil {
		noOverrideIds = computeFieldMeta(def).noOverrideIds
	}

	merged := make(happydns.CheckerOptions)
	// positionals are returned in order of increasing specificity.
	for _, p := range positionals {
		for k, v := range p.Options {
			// If the key is NoOverride and already set by a less specific scope, skip it.
			if noOverrideIds[k] {
				if _, exists := merged[k]; exists {
					continue
				}
			}
			merged[k] = v
		}
	}
	return merged, nil
}

// BuildMergedCheckerOptions merges stored options with runtime overrides.
// RunOpts are applied last and win over all stored levels.
func BuildMergedCheckerOptions(storedOpts happydns.CheckerOptions, runOpts happydns.CheckerOptions) happydns.CheckerOptions {
	result := make(happydns.CheckerOptions)
	maps.Copy(result, storedOpts)
	maps.Copy(result, runOpts)
	return result
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
	// Determine which field IDs are auto-filled or NoOverride for this checker.
	var autoFillIds map[string]string
	var noOverrideScopes map[string]happydns.CheckScopeType
	if def := checkerPkg.FindChecker(checkerName); def != nil {
		meta := computeFieldMeta(def)
		autoFillIds = meta.autoFillIds
		noOverrideScopes = meta.noOverrideScopes
	}

	currentScope := scopeFromIdentifiers(userId, domainId, serviceId)

	filtered := make(happydns.CheckerOptions, len(opts))
	for k, v := range opts {
		if isEmptyValue(v) || autoFillIds[k] != "" {
			continue
		}
		// Defense-in-depth: strip NoOverride fields at scopes below their definition.
		if defScope, ok := noOverrideScopes[k]; ok && currentScope > defScope {
			continue
		}
		filtered[k] = v
	}
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

	// Determine NoOverride scopes for defense-in-depth stripping.
	var noOverrideScopes map[string]happydns.CheckScopeType
	if def := checkerPkg.FindChecker(checkerName); def != nil {
		noOverrideScopes = computeFieldMeta(def).noOverrideScopes
	}
	currentScope := scopeFromIdentifiers(userId, domainId, serviceId)

	for k, v := range newOpts {
		// Defense-in-depth: skip NoOverride fields at scopes below their definition.
		if defScope, ok := noOverrideScopes[k]; ok && currentScope > defScope {
			continue
		}
		if isEmptyValue(v) {
			delete(existing, k)
		} else {
			existing[k] = v
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

// collectFieldsForScope returns the fields from a CheckerOptionsDocumentation
// that are valid at the given scope level. RunOpts are never included for
// persisted scopes.
func collectFieldsForScope(doc happydns.CheckerOptionsDocumentation, scope happydns.CheckScopeType) []happydns.CheckerOptionDocumentation {
	var fields []happydns.CheckerOptionDocumentation
	switch scope {
	case happydns.CheckScopeAdmin:
		fields = append(fields, doc.AdminOpts...)
	case happydns.CheckScopeUser:
		fields = append(fields, doc.UserOpts...)
	case happydns.CheckScopeDomain, happydns.CheckScopeZone:
		fields = append(fields, doc.DomainOpts...)
	case happydns.CheckScopeService:
		fields = append(fields, doc.ServiceOpts...)
	}
	return fields
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

	// Collect fields for this scope from the checker definition.
	// When withRunOpts is true (trigger time), also include all persisted-scope
	// fields so that options already stored at a different scope level (e.g.
	// admin-level options merged into the final opts map) are not rejected as
	// unknown.
	var allFields []happydns.CheckerOptionDocumentation
	if withRunOpts {
		allFields = append(allFields, def.Options.AdminOpts...)
		allFields = append(allFields, def.Options.UserOpts...)
		allFields = append(allFields, def.Options.DomainOpts...)
		allFields = append(allFields, def.Options.ServiceOpts...)
		allFields = append(allFields, def.Options.RunOpts...)
	} else {
		allFields = collectFieldsForScope(def.Options, scope)
	}

	// Collect fields from rules that declare their own options at this scope.
	for _, rule := range def.Rules {
		if rwo, ok := rule.(happydns.CheckRuleWithOptions); ok {
			ruleDoc := rwo.Options()
			if withRunOpts {
				allFields = append(allFields, ruleDoc.AdminOpts...)
				allFields = append(allFields, ruleDoc.UserOpts...)
				allFields = append(allFields, ruleDoc.DomainOpts...)
				allFields = append(allFields, ruleDoc.ServiceOpts...)
				allFields = append(allFields, ruleDoc.RunOpts...)
			} else {
				allFields = append(allFields, collectFieldsForScope(ruleDoc, scope)...)
			}
		}
	}

	// Filter out auto-fill fields: they are system-provided at runtime
	// and should not be validated against user input.
	autoFillIds := computeFieldMeta(def).autoFillIds
	var validatableFields []happydns.CheckerOptionDocumentation
	for _, f := range allFields {
		if _, isAutoFill := autoFillIds[f.Id]; !isAutoFill {
			validatableFields = append(validatableFields, f)
		}
	}

	// Validate against field definitions. ValidateMapValues lives in the
	// forms package and works with happydns.Field; CheckerOptionDocumentation
	// is structurally identical so an element-wise conversion is enough.
	if len(validatableFields) > 0 {
		asFields := make([]happydns.Field, len(validatableFields))
		for i, opt := range validatableFields {
			asFields[i] = happydns.FieldFromCheckerOption(opt)
		}
		if err := forms.ValidateMapValues(opts, asFields); err != nil {
			return err
		}
	}

	// Check if any rule implements OptionsValidator.
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
	// Defense-in-depth: reject NoOverride fields at scopes below their definition.
	if def := checkerPkg.FindChecker(checkerName); def != nil {
		meta := computeFieldMeta(def)
		if defScope, ok := meta.noOverrideScopes[optName]; ok {
			currentScope := scopeFromIdentifiers(userId, domainId, serviceId)
			if currentScope > defScope {
				return fmt.Errorf("option %q cannot be overridden at this scope level", optName)
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
	}

	scanDoc := func(doc happydns.CheckerOptionsDocumentation) {
		type scopedGroup struct {
			fields []happydns.CheckerOptionDocumentation
			scope  happydns.CheckScopeType
		}
		groups := []scopedGroup{
			{doc.AdminOpts, happydns.CheckScopeAdmin},
			{doc.UserOpts, happydns.CheckScopeUser},
			{doc.DomainOpts, happydns.CheckScopeDomain},
			{doc.ServiceOpts, happydns.CheckScopeService},
			{doc.RunOpts, happydns.CheckScopeService}, // RunOpts have no distinct scope; use Service as ceiling.
		}
		for _, g := range groups {
			for _, f := range g.fields {
				if f.AutoFill != "" {
					meta.autoFillIds[f.Id] = f.AutoFill
				}
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
					continue
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

// resolveAutoFill looks up the checker definition, scans its fields for AutoFill
// attributes, builds the execution context from storage, and returns a map of
// field ID to resolved value. Returns an empty map (not nil) when there is
// nothing to fill.
func (u *CheckerOptionsUsecase) resolveAutoFill(
	checkerName string,
	target happydns.CheckTarget,
) (happydns.CheckerOptions, error) {
	def := checkerPkg.FindChecker(checkerName)
	if def == nil {
		return make(happydns.CheckerOptions), nil
	}

	autoFillFields := computeFieldMeta(def).autoFillIds
	if len(autoFillFields) == 0 {
		return make(happydns.CheckerOptions), nil
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
	return result, nil
}

// BuildMergedCheckerOptionsWithAutoFill merges stored options, runtime overrides,
// and auto-fill values. Auto-fill values are applied last and always win.
func (u *CheckerOptionsUsecase) BuildMergedCheckerOptionsWithAutoFill(
	checkerName string,
	userId *happydns.Identifier,
	domainId *happydns.Identifier,
	serviceId *happydns.Identifier,
	runOpts happydns.CheckerOptions,
) (happydns.CheckerOptions, error) {
	positionals, err := u.store.GetCheckerConfiguration(checkerName, userId, domainId, serviceId)
	if err != nil {
		return nil, err
	}

	def := checkerPkg.FindChecker(checkerName)

	// Merge stored options from least to most specific, respecting NoOverride.
	var meta checkerFieldMeta
	if def != nil {
		meta = computeFieldMeta(def)
	}

	storedOpts := make(happydns.CheckerOptions)
	for _, p := range positionals {
		for k, v := range p.Options {
			if meta.noOverrideIds[k] {
				if _, exists := storedOpts[k]; exists {
					continue
				}
			}
			storedOpts[k] = v
		}
	}

	// Apply runtime overrides on top.
	merged := BuildMergedCheckerOptions(storedOpts, runOpts)

	// Restore NoOverride fields from storedOpts so that runOpts cannot override them.
	for id := range meta.noOverrideIds {
		if v, ok := storedOpts[id]; ok {
			merged[id] = v
		}
	}

	// Resolve auto-fill values (always win).
	if def != nil && len(meta.autoFillIds) > 0 {
		target := happydns.CheckTarget{
			UserId:    happydns.FormatIdentifier(userId),
			DomainId:  happydns.FormatIdentifier(domainId),
			ServiceId: happydns.FormatIdentifier(serviceId),
		}
		ctx, err := u.buildAutoFillContext(target)
		if err != nil {
			return nil, err
		}
		for fieldId, autoFillKey := range meta.autoFillIds {
			if val, ok := ctx[autoFillKey]; ok {
				merged[fieldId] = val
			}
		}
	}

	return merged, nil
}
