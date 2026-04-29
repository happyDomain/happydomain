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

// Keep the set small — anything exotic falls through to the raw JSON editor.
export type FieldKind = "text" | "url" | "secret" | "headers";

export interface ChannelConfigField {
    key: string;
    kind: FieldKind;
    required?: boolean;
    i18nLabel: string;
    i18nHelp?: string;
}

export interface ChannelConfigSchema {
    fields: ChannelConfigField[];
}

// Field names mirror JSON tags in internal/notification/*_sender.go.
export const CHANNEL_CONFIG_SCHEMAS: Record<string, ChannelConfigSchema> = {
    email: {
        fields: [
            {
                key: "emailAddress",
                kind: "text",
                i18nLabel: "settings.notifications.channels.fields.emailAddress",
                i18nHelp: "settings.notifications.channels.fields.emailAddressHelp",
            },
        ],
    },
    webhook: {
        fields: [
            {
                key: "webhookUrl",
                kind: "url",
                required: true,
                i18nLabel: "settings.notifications.channels.fields.webhookUrl",
            },
            {
                key: "webhookHeaders",
                kind: "headers",
                i18nLabel: "settings.notifications.channels.fields.webhookHeaders",
                i18nHelp: "settings.notifications.channels.fields.webhookHeadersHelp",
            },
            {
                key: "webhookSecret",
                kind: "secret",
                i18nLabel: "settings.notifications.channels.fields.webhookSecret",
                i18nHelp: "settings.notifications.channels.fields.webhookSecretHelp",
            },
        ],
    },
    unifiedpush: {
        fields: [
            {
                key: "unifiedPushEndpoint",
                kind: "url",
                required: true,
                i18nLabel: "settings.notifications.channels.fields.unifiedPushEndpoint",
                i18nHelp: "settings.notifications.channels.fields.unifiedPushEndpointHelp",
            },
        ],
    },
};

export function getChannelConfigSchema(type: string | undefined): ChannelConfigSchema | undefined {
    if (!type) return undefined;
    return CHANNEL_CONFIG_SCHEMAS[type];
}

export function emptyConfigForSchema(schema: ChannelConfigSchema): Record<string, unknown> {
    const cfg: Record<string, unknown> = {};
    for (const f of schema.fields) {
        if (f.kind === "headers") cfg[f.key] = {};
        else cfg[f.key] = "";
    }
    return cfg;
}
