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
    getPluginsTests,
    getPluginsTestsByPid,
    getPluginsTestsByPidOptions,
    postPluginsTestsByPidOptions,
    putPluginsTestsByPidOptions,
    getPluginsTestsByPidOptionsByOptname,
    putPluginsTestsByPidOptionsByOptname,
} from "$lib/api-base/sdk.gen";
import { unwrapSdkResponse } from "./errors";
import type {
    PluginList,
    PluginStatus,
    PluginOptions,
} from "$lib/model/plugin";

export async function listPlugins(): Promise<PluginList> {
    return unwrapSdkResponse(await getPluginsTests()) as PluginList;
}

export async function getPluginStatus(pluginId: string): Promise<PluginStatus> {
    return unwrapSdkResponse(
        await getPluginsTestsByPid({
            path: { pid: pluginId },
        }),
    ) as PluginStatus;
}

export async function getPluginOptions(pluginId: string): Promise<PluginOptions> {
    return unwrapSdkResponse(
        await getPluginsTestsByPidOptions({
            path: { pid: pluginId },
        }),
    ) as PluginOptions;
}

export async function addPluginOptions(pluginId: string, options: PluginOptions): Promise<boolean> {
    return unwrapSdkResponse(
        await postPluginsTestsByPidOptions({
            path: { pid: pluginId },
            body: { options } as any,
        }),
    ) as boolean;
}

export async function updatePluginOptions(pluginId: string, options: PluginOptions): Promise<boolean> {
    return unwrapSdkResponse(
        await putPluginsTestsByPidOptions({
            path: { pid: pluginId },
            body: { options } as any,
        }),
    ) as boolean;
}

export async function getPluginOption(pluginId: string, optionName: string): Promise<any> {
    return unwrapSdkResponse(
        await getPluginsTestsByPidOptionsByOptname({
            path: { pid: pluginId, optname: optionName },
        }),
    );
}

export async function setPluginOption(pluginId: string, optionName: string, value: any): Promise<boolean> {
    return unwrapSdkResponse(
        await putPluginsTestsByPidOptionsByOptname({
            path: { pid: pluginId, optname: optionName },
            body: value as any,
        }),
    ) as boolean;
}
