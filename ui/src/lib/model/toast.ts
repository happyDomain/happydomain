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
    onclick: () => void;

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
