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

import { writable } from 'svelte/store';
import { Toast, type NewToast } from '$lib/model/toast';

function createToastsStore() {
    const { subscribe, update } = writable([]);

    const addToast = (o: NewToast) => {
        const toast = new Toast(o, dismiss);

        update((all: any) => {
            all.unshift(toast);
            return all;
        })
    };

    const addErrorToast = (o: NewToast) => {
        if (!o.title) o.title = 'An error occured!';
        if (!o.type) o.type = 'error';

        return addToast(o);
    };

    const dismiss = (id: string) => {
        update((all: any) => all.filter((t: any) => t.id !== id))
    }

    return {
        subscribe,

        addToast,
        addErrorToast,

        dismiss,
  };

}

export const toasts: any = createToastsStore()
