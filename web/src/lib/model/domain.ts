// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

export interface ZoneHistory {
    id: string;
    id_author: string;
    default_ttl: number;
    last_modified: Date;
    commit_message: string;
    commit_date: Date;
    published?: Date;
}

export interface Domain {
    id: string;
    id_owner: string;
    id_provider: string;
    domain: string;
    group: string;
    zone_history: Array<string>;
    zone_meta?: Array<ZoneHistory>;

    // interface property
    wait: boolean;
}

export interface DomainLog {
    id: string;
    id_user: string;
    date: Date;
    content: string;
    level: number;
}
