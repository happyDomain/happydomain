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

import {
    getNotificationsChannelTypes,
    getNotificationsChannels,
    postNotificationsChannels,
    getNotificationsChannelsByChannelId,
    putNotificationsChannelsByChannelId,
    deleteNotificationsChannelsByChannelId,
    postNotificationsChannelsByChannelIdTest,
    getNotificationsPreferences,
    postNotificationsPreferences,
    getNotificationsPreferencesByPrefId,
    putNotificationsPreferencesByPrefId,
    deleteNotificationsPreferencesByPrefId,
    getNotificationsHistory,
} from "$lib/api-base/sdk.gen";
import type {
    HappydnsNotificationChannel,
    HappydnsNotificationChannelWritable,
    HappydnsNotificationPreference,
    HappydnsNotificationPreferenceWritable,
    HappydnsNotificationRecord,
} from "$lib/api-base/types.gen";
import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

// Extend Writable types with Config — OpenAPI models it as free-form so the generated types omit it.
export type NotificationChannelInput = HappydnsNotificationChannelWritable & {
    config?: Record<string, unknown>;
};
export type NotificationChannel = HappydnsNotificationChannel & {
    config?: Record<string, unknown>;
};

export type NotificationPreferenceInput = HappydnsNotificationPreferenceWritable;
export type NotificationPreference = HappydnsNotificationPreference;

export type NotificationRecord = HappydnsNotificationRecord;

export async function listChannelTypes(): Promise<string[]> {
    return unwrapSdkResponse(await getNotificationsChannelTypes()) as string[];
}

export async function listChannels(): Promise<NotificationChannel[]> {
    return unwrapSdkResponse(await getNotificationsChannels()) as NotificationChannel[];
}

export async function getChannel(id: string): Promise<NotificationChannel> {
    return unwrapSdkResponse(
        await getNotificationsChannelsByChannelId({ path: { channelId: id } }),
    ) as NotificationChannel;
}

export async function createChannel(
    channel: NotificationChannelInput,
): Promise<NotificationChannel> {
    return unwrapSdkResponse(
        await postNotificationsChannels({
            body: channel as HappydnsNotificationChannelWritable,
        }),
    ) as NotificationChannel;
}

export async function updateChannel(
    id: string,
    channel: NotificationChannelInput,
): Promise<NotificationChannel> {
    return unwrapSdkResponse(
        await putNotificationsChannelsByChannelId({
            path: { channelId: id },
            body: channel as HappydnsNotificationChannelWritable,
        }),
    ) as NotificationChannel;
}

export async function deleteChannel(id: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await deleteNotificationsChannelsByChannelId({ path: { channelId: id } }),
    );
}

export async function testChannel(id: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await postNotificationsChannelsByChannelIdTest({ path: { channelId: id } }),
    );
}

export async function listPreferences(): Promise<NotificationPreference[]> {
    return unwrapSdkResponse(await getNotificationsPreferences()) as NotificationPreference[];
}

export async function getPreference(id: string): Promise<NotificationPreference> {
    return unwrapSdkResponse(
        await getNotificationsPreferencesByPrefId({ path: { prefId: id } }),
    ) as NotificationPreference;
}

export async function createPreference(
    pref: NotificationPreferenceInput,
): Promise<NotificationPreference> {
    return unwrapSdkResponse(
        await postNotificationsPreferences({ body: pref }),
    ) as NotificationPreference;
}

export async function updatePreference(
    id: string,
    pref: NotificationPreferenceInput,
): Promise<NotificationPreference> {
    return unwrapSdkResponse(
        await putNotificationsPreferencesByPrefId({
            path: { prefId: id },
            body: pref,
        }),
    ) as NotificationPreference;
}

export async function deletePreference(id: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await deleteNotificationsPreferencesByPrefId({ path: { prefId: id } }),
    );
}

export async function listHistory(limit?: number): Promise<NotificationRecord[]> {
    return unwrapSdkResponse(
        await getNotificationsHistory({ query: limit ? { limit } : undefined }),
    ) as NotificationRecord[];
}
