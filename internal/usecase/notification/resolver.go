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

package notification

import (
	"log"

	"git.happydns.org/happyDomain/model"
)

// Read-only, safe to share between goroutines.
type Resolver struct {
	channelStore NotificationChannelStorage
	prefStore    NotificationPreferenceStorage
}

func NewResolver(channelStore NotificationChannelStorage, prefStore NotificationPreferenceStorage) *Resolver {
	return &Resolver{channelStore: channelStore, prefStore: prefStore}
}

// Specificity service > domain > global; falls back to DefaultNotificationPreference so opt-in defaults flow through.
func (r *Resolver) ResolvePreference(user *happydns.User, target happydns.CheckTarget) *happydns.NotificationPreference {
	prefs, err := r.prefStore.ListPreferencesByUser(user.Id)
	if err != nil {
		log.Printf("notification: failed to load preferences for user %q: %v", user.Id, err)
		return happydns.DefaultNotificationPreference()
	}

	var best *happydns.NotificationPreference
	bestSpecificity := -1
	for _, p := range prefs {
		s := p.MatchesTarget(target)
		if s > bestSpecificity {
			best = p
			bestSpecificity = s
		}
	}
	if best == nil {
		return happydns.DefaultNotificationPreference()
	}
	return best
}

func (r *Resolver) ResolveChannels(user *happydns.User, pref *happydns.NotificationPreference) []*happydns.NotificationChannel {
	allChannels, err := r.channelStore.ListChannelsByUser(user.Id)
	if err != nil {
		log.Printf("notification: failed to load channels for user %q: %v", user.Id, err)
		return nil
	}

	var allowed map[string]bool
	if len(pref.ChannelIds) > 0 {
		allowed = make(map[string]bool, len(pref.ChannelIds))
		for _, id := range pref.ChannelIds {
			allowed[id.String()] = true
		}
	}

	var result []*happydns.NotificationChannel
	for _, ch := range allChannels {
		if !ch.Enabled {
			continue
		}
		if allowed != nil && !allowed[ch.Id.String()] {
			continue
		}
		result = append(result, ch)
	}
	return result
}
