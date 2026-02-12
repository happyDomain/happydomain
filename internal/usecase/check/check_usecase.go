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

package check

import (
	"cmp"
	"fmt"
	"log"
	"maps"
	"slices"

	"git.happydns.org/happyDomain/checks"
	"git.happydns.org/happyDomain/model"
)

type checkerUsecase struct {
	config        *happydns.Options
	store         CheckerStorage
	autoFillStore CheckAutoFillStorage
}

func NewCheckerUsecase(cfg *happydns.Options, store CheckerStorage, autoFillStore CheckAutoFillStorage) happydns.CheckerUsecase {
	return &checkerUsecase{
		config:        cfg,
		store:         store,
		autoFillStore: autoFillStore,
	}
}

func (tu *checkerUsecase) GetChecker(cname string) (happydns.Checker, error) {
	checker, err := checks.FindChecker(cname)
	if err != nil {
		return nil, fmt.Errorf("unable to find check named %q: %w", cname, err)
	}

	return checker, nil
}

// copyNonEmpty copies key/value pairs from src into dst, skipping nil or empty-string values.
func copyNonEmpty(dst, src happydns.CheckerOptions) {
	for k, v := range src {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		dst[k] = v
	}
}

func compareIdentifiers(a, b *happydns.Identifier) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	if a.Equals(*b) {
		return 0
	}

	return a.Compare(*b)
}

// CompareCheckerOptionsPositional defines the merge precedence ordering for
// checker option configs: admin < user < domain < service.
func CompareCheckerOptionsPositional(a, b *happydns.CheckerOptionsPositional) int {
	if a.CheckName != b.CheckName {
		return cmp.Compare(a.CheckName, b.CheckName)
	}
	if res := compareIdentifiers(a.UserId, b.UserId); res != 0 {
		return res
	}
	if res := compareIdentifiers(a.DomainId, b.DomainId); res != 0 {
		return res
	}
	return compareIdentifiers(a.ServiceId, b.ServiceId)
}

func (tu *checkerUsecase) GetCheckerOptions(cname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) (*happydns.CheckerOptions, error) {
	configs, err := tu.store.GetCheckerConfiguration(cname, userid, domainid, serviceid)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(configs, CompareCheckerOptionsPositional)

	opts := make(happydns.CheckerOptions)

	for _, c := range configs {
		maps.Copy(opts, c.Options)
	}

	return &opts, nil
}

func (tu *checkerUsecase) ListCheckers() (*map[string]happydns.Checker, error) {
	return checks.GetCheckers(), nil
}

// GetStoredCheckerOptionsNoDefault returns the stored options (user/domain/service scopes)
// with auto-fill variables applied, but without checker-defined defaults or run-time overrides.
func (tu *checkerUsecase) GetStoredCheckerOptionsNoDefault(cname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) (happydns.CheckerOptions, error) {
	stored, err := tu.GetCheckerOptions(cname, userid, domainid, serviceid)
	if err != nil {
		return nil, err
	}

	var opts happydns.CheckerOptions
	if stored != nil {
		opts = *stored
	} else {
		opts = make(happydns.CheckerOptions)
	}

	checker, err := tu.GetChecker(cname)
	if err != nil {
		return opts, nil
	}

	return tu.applyAutoFill(checker, userid, domainid, serviceid, opts), nil
}

// BuildMergedCheckerOptions merges checker options from all sources in priority order:
// checker defaults < stored (user/domain/service) options < runOpts < auto-fill variables.
func (tu *checkerUsecase) BuildMergedCheckerOptions(cname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, runOpts happydns.CheckerOptions) (happydns.CheckerOptions, error) {
	merged := make(happydns.CheckerOptions)

	// 1. Fill checker defaults.
	checker, err := tu.GetChecker(cname)
	if err != nil {
		log.Printf("Warning: unable to get checker %q for default options: %v", cname, err)
	} else {
		opts := checker.Options()

		allOpts := []happydns.CheckerOptionDocumentation{}
		allOpts = append(allOpts, opts.RunOpts...)
		allOpts = append(allOpts, opts.ServiceOpts...)
		allOpts = append(allOpts, opts.DomainOpts...)
		allOpts = append(allOpts, opts.UserOpts...)
		allOpts = append(allOpts, opts.AdminOpts...)
		for _, opt := range allOpts {
			if opt.Default != nil {
				merged[opt.Id] = opt.Default
			} else if opt.Placeholder != "" {
				merged[opt.Id] = opt.Placeholder
			}
		}
	}

	// 2. Override with stored options (user/domain/service scopes).
	baseOptions, err := tu.GetCheckerOptions(cname, userid, domainid, serviceid)
	if err != nil {
		return merged, fmt.Errorf("could not fetch stored checker options for %s: %w", cname, err)
	}
	if baseOptions != nil {
		copyNonEmpty(merged, *baseOptions)
	}

	// 3. Override with caller-supplied run options.
	copyNonEmpty(merged, runOpts)

	// 4. Inject auto-fill variables (always win over any user-supplied value).
	if checker != nil {
		merged = tu.applyAutoFill(checker, userid, domainid, serviceid, merged)
	}

	return merged, nil
}

// applyAutoFill resolves auto-fill fields declared by the checker and injects
// the context-resolved values into a copy of opts.
func (tu *checkerUsecase) applyAutoFill(
	checker happydns.Checker,
	userid *happydns.Identifier,
	domainid *happydns.Identifier,
	serviceid *happydns.Identifier,
	opts happydns.CheckerOptions,
) happydns.CheckerOptions {
	// Collect which auto-fill keys are needed.
	needed := make(map[string]string) // autoFill constant → field id
	options := checker.Options()
	for _, groups := range [][]happydns.CheckerOptionDocumentation{
		options.RunOpts, options.DomainOpts, options.ServiceOpts,
		options.UserOpts, options.AdminOpts,
	} {
		for _, opt := range groups {
			if opt.AutoFill != "" {
				needed[opt.AutoFill] = opt.Id
			}
		}
	}

	if len(needed) == 0 || tu.autoFillStore == nil {
		return opts
	}

	autoFillCtx := tu.buildAutoFillContext(userid, domainid, serviceid)

	result := maps.Clone(opts)
	for autoFillKey, fieldId := range needed {
		if val, ok := autoFillCtx[autoFillKey]; ok {
			result[fieldId] = val
		}
	}
	return result
}

// buildAutoFillContext resolves the available auto-fill values for the given
// user/domain/service identifiers.
func (tu *checkerUsecase) buildAutoFillContext(userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) map[string]string {
	ctx := make(map[string]string)

	if domainid != nil {
		if domain, err := tu.autoFillStore.GetDomain(*domainid); err == nil {
			ctx[happydns.AutoFillDomainName] = domain.DomainName
		}
	} else if serviceid != nil && userid != nil {
		// To resolve service context we need to find which domain/zone owns the service.
		user, err := tu.autoFillStore.GetUser(*userid)
		if err != nil {
			return ctx
		}
		domains, err := tu.autoFillStore.ListDomains(user)
		if err != nil {
			return ctx
		}
		for _, domain := range domains {
			if len(domain.ZoneHistory) == 0 {
				continue
			}
			// The first element in ZoneHistory is the current (most recent) zone.
			zoneMsg, err := tu.autoFillStore.GetZone(domain.ZoneHistory[0])
			if err != nil {
				continue
			}
			for subdomain, svcs := range zoneMsg.Services {
				for _, svc := range svcs {
					if svc.Id.Equals(*serviceid) {
						ctx[happydns.AutoFillDomainName] = domain.DomainName
						ctx[happydns.AutoFillSubdomain] = string(subdomain)
						ctx[happydns.AutoFillServiceType] = svc.Type
						return ctx
					}
				}
			}
		}
	}

	return ctx
}

func (tu *checkerUsecase) SetCheckerOptions(cname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.CheckerOptions) error {
	// filter opts that correspond to the level set
	checker, err := tu.GetChecker(cname)
	if err != nil {
		return fmt.Errorf("unable to get checker: %w", err)
	}

	options := checker.Options()

	var relevantOpts []happydns.CheckerOptionDocumentation
	if serviceid != nil {
		relevantOpts = options.ServiceOpts
	} else if domainid != nil {
		relevantOpts = options.DomainOpts
	} else if userid != nil {
		relevantOpts = options.UserOpts
	} else {
		relevantOpts = options.AdminOpts
	}

	allowed := make(map[string]struct{}, len(relevantOpts))
	for _, opt := range relevantOpts {
		allowed[opt.Id] = struct{}{}
	}

	filteredOpts := make(happydns.CheckerOptions)
	for id := range allowed {
		if val, exists := opts[id]; exists && val != "" {
			filteredOpts[id] = val
		}
	}

	return tu.store.UpdateCheckerConfiguration(cname, userid, domainid, serviceid, filteredOpts)
}

func (tu *checkerUsecase) OverwriteSomeCheckerOptions(cname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.CheckerOptions) error {
	current, err := tu.GetCheckerOptions(cname, userid, domainid, serviceid)
	if err != nil {
		return err
	}

	maps.Copy(*current, opts)

	return tu.store.UpdateCheckerConfiguration(cname, userid, domainid, serviceid, *current)
}
