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

package plugin

import (
	"fmt"
	"log"
	"maps"
	"sort"

	"git.happydns.org/happyDomain/model"
)

type testPluginUsecase struct {
	config        *happydns.Options
	manager       happydns.PluginManager
	store         PluginStorage
	autoFillStore PluginAutoFillStorage
}

func NewTestPluginUsecase(cfg *happydns.Options, manager happydns.PluginManager, store PluginStorage, autoFillStore PluginAutoFillStorage) happydns.TestPluginUsecase {
	return &testPluginUsecase{
		config:        cfg,
		manager:       manager,
		store:         store,
		autoFillStore: autoFillStore,
	}
}

func (tu *testPluginUsecase) GetTestPlugin(pname string) (happydns.TestPlugin, error) {
	plugin, ok := tu.manager.GetTestPlugin(pname)
	if !ok {
		return nil, fmt.Errorf("unable to find plugin named %q", pname)
	} else {
		return plugin, nil
	}
}

type ByOptionPosition []*happydns.PluginOptionsPositional

func (a ByOptionPosition) Len() int      { return len(a) }
func (a ByOptionPosition) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByOptionPosition) Less(i, j int) bool {
	if a[i].PluginName != a[j].PluginName {
		return a[i].PluginName < a[j].PluginName
	}

	if res := compareIdentifiers(a[i].UserId, a[j].UserId); res != 0 {
		return res < 0
	}

	if res := compareIdentifiers(a[i].DomainId, a[j].DomainId); res != 0 {
		return res < 0
	}

	if res := compareIdentifiers(a[i].ServiceId, a[j].ServiceId); res != 0 {
		return res < 0
	}

	return false
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

func (tu *testPluginUsecase) GetTestPluginOptions(pname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) (*happydns.PluginOptions, error) {
	configs, err := tu.store.GetPluginConfiguration(pname, userid, domainid, serviceid)
	if err != nil {
		return nil, err
	}

	sort.Sort(ByOptionPosition(configs))

	opts := make(happydns.PluginOptions)

	for _, c := range configs {
		maps.Copy(opts, c.Options)
	}

	return &opts, nil
}

func (tu *testPluginUsecase) ListTestPlugins() ([]happydns.TestPlugin, error) {
	return tu.manager.GetTestPlugins(), nil
}

// GetStoredTestPluginOptionsNoDefault returns the stored options (user/domain/service scopes)
// with auto-fill variables applied, but without plugin-defined defaults or run-time overrides.
func (tu *testPluginUsecase) GetStoredTestPluginOptionsNoDefault(pname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) (happydns.PluginOptions, error) {
	stored, err := tu.GetTestPluginOptions(pname, userid, domainid, serviceid)
	if err != nil {
		return nil, err
	}

	var opts happydns.PluginOptions
	if stored != nil {
		opts = *stored
	} else {
		opts = make(happydns.PluginOptions)
	}

	plugin, err := tu.GetTestPlugin(pname)
	if err != nil {
		return opts, nil
	}

	return tu.applyAutoFill(plugin, userid, domainid, serviceid, opts), nil
}

// BuildMergedTestPluginOptions merges plugin options from all sources in priority order:
// plugin defaults < stored (user/domain/service) options < runOpts < auto-fill variables.
func (tu *testPluginUsecase) BuildMergedTestPluginOptions(pname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, runOpts happydns.PluginOptions) (happydns.PluginOptions, error) {
	merged := make(happydns.PluginOptions)

	// 1. Fill plugin defaults.
	plugin, err := tu.GetTestPlugin(pname)
	if err != nil {
		log.Printf("Warning: unable to get plugin %q for default options: %v", pname, err)
	} else {
		availableOpts := plugin.AvailableOptions()
		allOpts := []happydns.PluginOptionDocumentation{}
		allOpts = append(allOpts, availableOpts.RunOpts...)
		allOpts = append(allOpts, availableOpts.ServiceOpts...)
		allOpts = append(allOpts, availableOpts.DomainOpts...)
		allOpts = append(allOpts, availableOpts.UserOpts...)
		allOpts = append(allOpts, availableOpts.AdminOpts...)
		for _, opt := range allOpts {
			if opt.Default != nil {
				merged[opt.Id] = opt.Default
			}
		}
	}

	// 2. Override with stored options (user/domain/service scopes).
	baseOptions, err := tu.GetTestPluginOptions(pname, userid, domainid, serviceid)
	if err != nil {
		return merged, fmt.Errorf("could not fetch stored plugin options for %s: %w", pname, err)
	}
	if baseOptions != nil {
		maps.Copy(merged, *baseOptions)
	}

	// 3. Override with caller-supplied run options.
	maps.Copy(merged, runOpts)

	// 4. Inject auto-fill variables (always win over any user-supplied value).
	if plugin != nil {
		merged = tu.applyAutoFill(plugin, userid, domainid, serviceid, merged)
	}

	return merged, nil
}

// applyAutoFill resolves auto-fill fields declared by the plugin and injects
// the context-resolved values into a copy of opts.
func (tu *testPluginUsecase) applyAutoFill(
	plugin happydns.TestPlugin,
	userid *happydns.Identifier,
	domainid *happydns.Identifier,
	serviceid *happydns.Identifier,
	opts happydns.PluginOptions,
) happydns.PluginOptions {
	allOpts := plugin.AvailableOptions()

	// Collect which auto-fill keys are needed.
	needed := make(map[string]string) // autoFill constant → field id
	for _, groups := range [][]happydns.PluginOptionDocumentation{
		allOpts.RunOpts, allOpts.DomainOpts, allOpts.ServiceOpts,
		allOpts.UserOpts, allOpts.AdminOpts,
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
func (tu *testPluginUsecase) buildAutoFillContext(userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) map[string]string {
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

func (tu *testPluginUsecase) SetTestPluginOptions(pname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.PluginOptions) error {
	// filter opts that correspond to the level set
	plugin, err := tu.GetTestPlugin(pname)
	if err != nil {
		return fmt.Errorf("unable to get test plugin: %w", err)
	}

	var optNames []string
	if serviceid != nil {
		for _, opt := range plugin.AvailableOptions().ServiceOpts {
			optNames = append(optNames, opt.Id)
		}
	} else if domainid != nil {
		for _, opt := range plugin.AvailableOptions().DomainOpts {
			optNames = append(optNames, opt.Id)
		}
	} else if userid != nil {
		for _, opt := range plugin.AvailableOptions().UserOpts {
			optNames = append(optNames, opt.Id)
		}
	} else {
		for _, opt := range plugin.AvailableOptions().AdminOpts {
			optNames = append(optNames, opt.Id)
		}
	}

	// Filter opts to only include keys that are in optNames
	filteredOpts := make(happydns.PluginOptions)
	for _, optName := range optNames {
		if val, exists := opts[optName]; exists && val != "" {
			filteredOpts[optName] = val
		}
	}

	return tu.store.UpdatePluginConfiguration(pname, userid, domainid, serviceid, filteredOpts)
}

func (tu *testPluginUsecase) OverwriteSomeTestPluginOptions(pname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.PluginOptions) error {
	current, err := tu.GetTestPluginOptions(pname, userid, domainid, serviceid)
	if err != nil {
		return err
	}

	maps.Copy(*current, opts)

	return tu.store.UpdatePluginConfiguration(pname, userid, domainid, serviceid, *current)
}
