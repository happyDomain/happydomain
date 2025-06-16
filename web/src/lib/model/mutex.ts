// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2025 happyDomain
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

export class Mutex {
    private _locked = false;
    private _waiting: Array<() => void> = [];

    async lock(): Promise<() => void> {
        const unlock = () => {
            const next = this._waiting.shift();
            if (next) {
                next();
            } else {
                this._locked = false;
            }
        };

        if (this._locked) {
            await new Promise<void>((resolve) => {
                this._waiting.push(() => resolve());
            });
        }

        this._locked = true;
        return unlock;
    }
}
