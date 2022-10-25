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

    const addErrorToast = (o: Toast) => {
        if (!o.title) o.title = 'Une erreur est survenue !';
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
