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

import type {
    HappydnsPluginAvailability,
    HappydnsPluginOptionDocumentation,
    HappydnsPluginOptionsDocumentation,
    HappydnsPluginOptions,
} from "$lib/api-base/types.gen";

// Re-export auto-generated types with better names
export type PluginAvailability = HappydnsPluginAvailability;
export type PluginOptions = HappydnsPluginOptions;
export type PluginOptionsDocumentation = HappydnsPluginOptionsDocumentation;

// Make 'id' required for PluginOptionDocumentation
export interface PluginOptionDocumentation extends Omit<HappydnsPluginOptionDocumentation, "id"> {
    id: string;
}

// Make 'name' and 'version' required for PluginVersionInfo
export interface PluginVersionInfo {
    name: string;
    version: string;
    availableOn?: PluginAvailability;
}

// Make 'name' and 'version' required for PluginStatus
export interface PluginStatus {
    name: string;
    version: string;
    availableOn?: PluginAvailability;
    options?: PluginOptionsDocumentation;
}

export type PluginList = Record<string, PluginVersionInfo>;
