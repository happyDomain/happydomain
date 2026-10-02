// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { describe, it, expect } from "vitest";

import { CHANNEL_CONFIG_SCHEMAS, emptyConfigForSchema } from "./channelConfigs";

// The form sends the config as it builds it: its keys are those the server
// decodes, the JSON tags of the configs in internal/notifier/*_sender.go.
const SERVER_CONFIG_KEYS: Record<string, string[]> = {
    email: ["address"], // EmailConfig
    webhook: ["url", "headers", "secret"], // WebhookConfig
    unifiedpush: ["endpoint"], // UnifiedPushConfig
};

describe("CHANNEL_CONFIG_SCHEMAS", () => {
    it("covers the channel types the server knows", () => {
        expect(Object.keys(CHANNEL_CONFIG_SCHEMAS).sort()).toEqual(
            Object.keys(SERVER_CONFIG_KEYS).sort(),
        );
    });

    for (const [type, keys] of Object.entries(SERVER_CONFIG_KEYS)) {
        it(`uses the keys the server decodes for ${type}`, () => {
            expect(CHANNEL_CONFIG_SCHEMAS[type].fields.map((f) => f.key)).toEqual(keys);
        });
    }
});

describe("emptyConfigForSchema", () => {
    it("starts a webhook with the keys the server decodes", () => {
        expect(emptyConfigForSchema(CHANNEL_CONFIG_SCHEMAS.webhook)).toEqual({
            url: "",
            headers: {},
            secret: "",
        });
    });
});
