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

package happydns_test

import (
	"encoding/json"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// Decoding a request into a clone must leave the original as it was.
func TestNotificationChannelClone(t *testing.T) {
	orig := &happydns.NotificationChannel{
		Id:     happydns.Identifier{0x01},
		UserId: happydns.Identifier{0x02},
		Type:   "webhook",
		Config: json.RawMessage(`{"url":"https://example.com/a/rather/long/path","secret":"kept"}`),
	}
	want, _ := json.Marshal(orig)

	c := orig.Clone()
	if err := json.Unmarshal([]byte(`{"id":"AwM","userId":"BAQ","config":{"url":"x"}}`), c); err != nil {
		t.Fatal(err)
	}
	c.Id[0], c.UserId[0] = 0xff, 0xff

	if got, _ := json.Marshal(orig); string(got) != string(want) {
		t.Errorf("original changed through its clone:\n got %s\nwant %s", got, want)
	}
}

func TestNotificationChannelCloneNil(t *testing.T) {
	c := (&happydns.NotificationChannel{}).Clone()
	if c.Config != nil || c.Id != nil {
		t.Errorf("Clone of an empty channel = %+v, want empty fields left nil", c)
	}
}

func TestNotificationPreferenceClone(t *testing.T) {
	domain, service := happydns.Identifier{0x03}, happydns.Identifier{0x04}
	start, end := 22, 7
	orig := &happydns.NotificationPreference{
		Id:         happydns.Identifier{0x01},
		UserId:     happydns.Identifier{0x02},
		DomainId:   &domain,
		ServiceId:  &service,
		ChannelIds: []happydns.Identifier{{0x05}, {0x06}},
		QuietStart: &start,
		QuietEnd:   &end,
		Timezone:   "Europe/Paris",
	}
	want, _ := json.Marshal(orig)

	c := orig.Clone()
	if err := json.Unmarshal([]byte(`{"domainId":"CQk","serviceId":"CQk","channelIds":["CQk"],"quietStart":5,"quietEnd":6}`), c); err != nil {
		t.Fatal(err)
	}
	(*c.DomainId)[0], (*c.ServiceId)[0] = 0xff, 0xff
	c.ChannelIds[0][0] = 0xff
	c.Id[0], c.UserId[0] = 0xff, 0xff

	if got, _ := json.Marshal(orig); string(got) != string(want) {
		t.Errorf("original changed through its clone:\n got %s\nwant %s", got, want)
	}
}

func TestNotificationPreferenceCloneNil(t *testing.T) {
	c := (&happydns.NotificationPreference{}).Clone()
	if c.DomainId != nil || c.ServiceId != nil || c.ChannelIds != nil || c.QuietStart != nil || c.QuietEnd != nil {
		t.Errorf("Clone of an empty preference = %+v, want empty fields left nil", c)
	}
}
