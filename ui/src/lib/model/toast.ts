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

import cuid from 'cuid';

import type Color from './color';

export interface NewToast {
    type?: "info" | "success" | "warning" | "error",
    title?: string,
    message?: string,
    timeout?: number | undefined
    onclick?: () => void
}

export class Toast implements NewToast {
    id: string = cuid();
    type: "info" | "success" | "warning" | "error" = "info";
    title: string = "";
    message: string = "";
    timeout: number | undefined = undefined;
    timeoutInterval: ReturnType<typeof setTimeout> | undefined = undefined;
    dismissFunc: (id: string) => void;
    onclick: undefined | (() => void) = undefined;

    constructor(obj: NewToast, dismiss: (id: string) => void) {
        if (obj.type !== undefined) this.type = obj.type;
        if (obj.title !== undefined) this.title = obj.title;
        if (obj.message !== undefined) this.message = obj.message;
        if (obj.onclick !== undefined) this.onclick = obj.onclick;
        this.timeout = obj.timeout;

        this.dismissFunc = dismiss;

        if (this.timeout)
            this.resume();
    }

    dismiss() {
        this.dismissFunc(this.id);
    }

    pause() {
        clearTimeout(this.timeoutInterval);
    }

    resume() {
        this.timeoutInterval = setTimeout(() => this.dismissFunc(this.id), this.timeout)
    }

    getColor(): Color {
        switch(this.type) {
            case "info":
                return "info";
            case "success":
                return "success";
            case "warning":
                return "warning";
            case "error":
                return "danger";
            default:
                return "secondary";
        }
    }
}
