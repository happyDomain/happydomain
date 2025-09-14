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

export class ServiceMeta {
    _svctype = $state<string>('');
    _id? = $state<string | undefined>(undefined);
    _ownerid? = $state<string | undefined>(undefined);
    _domain = $state<string>('');
    _ttl? = $state<number | undefined>(undefined);
    _comment? = $state<string | undefined>(undefined);
    _mycomment? = $state<string | undefined>(undefined);
    _aliases? = $state<Array<string> | undefined>(undefined);
    _tmp_hint_nb? = $state<number | undefined>(undefined);

    constructor(init?: {
        _svctype: string;
        _id?: string;
        _ownerid?: string;
        _domain: string;
        _ttl?: number;
        _comment?: string;
        _mycomment?: string;
        _aliases?: Array<string>;
        _tmp_hint_nb?: number;
    }) {
        if (init) {
            this._svctype = init._svctype;
            this._domain = init._domain;
            if (init._id !== undefined) this._id = init._id;
            if (init._ownerid !== undefined) this._ownerid = init._ownerid;
            if (init._ttl !== undefined) this._ttl = init._ttl;
            if (init._comment !== undefined) this._comment = init._comment;
            if (init._mycomment !== undefined) this._mycomment = init._mycomment;
            if (init._aliases !== undefined) this._aliases = init._aliases;
            if (init._tmp_hint_nb !== undefined) this._tmp_hint_nb = init._tmp_hint_nb;
        }
    }
}

export class ServiceCombined extends ServiceMeta {
    Service = $state<any>(null);

    constructor(init?: {
        _svctype: string;
        _id?: string;
        _ownerid?: string;
        _domain: string;
        _ttl?: number;
        _comment?: string;
        _mycomment?: string;
        _aliases?: Array<string>;
        _tmp_hint_nb?: number;
        Service?: any;
    }) {
        super(init);
        if (init?.Service !== undefined) {
            this.Service = init.Service;
        }
    }
}
