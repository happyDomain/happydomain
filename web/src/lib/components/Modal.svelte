<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<!--
     sveltestrap's Modal leaves keyboard focus on the page behind the dialog, so
     Tab walks through the controls hidden under the overlay. This wrapper adds
     the missing focus management on top of it; use it instead of importing
     Modal from @sveltestrap/sveltestrap directly.
-->

<script lang="ts">
    import type { Snippet } from "svelte";

    import { Modal } from "@sveltestrap/sveltestrap";

    import { trapFocus } from "$lib/utils/focustrap";
    import { randomUUID } from "$lib/utils/uuid";

    interface Props {
        isOpen?: boolean;
        onopen?: () => void;
        children?: Snippet;
        [key: string]: unknown;
    }

    let { isOpen = false, onopen, children, ...rest }: Props = $props();

    const wrapperId = `modal-${randomUUID()}`;

    $effect(() => {
        if (!isOpen) return;

        let release: (() => void) | undefined;
        let frame: number | undefined;

        // The dialog is rendered into a portal appended to <body>, one tick
        // after isOpen flips and behind a transition, so wait for it to show up.
        const attach = (attempts: number) => {
            frame = undefined;

            const dialog = document
                .getElementById(wrapperId)
                ?.querySelector<HTMLElement>('.modal[role="dialog"]');

            if (dialog) {
                release = trapFocus(dialog, wrapperId);
            } else if (attempts > 0) {
                frame = requestAnimationFrame(() => attach(attempts - 1));
            }
        };
        attach(30);

        return () => {
            if (frame !== undefined) cancelAnimationFrame(frame);
            release?.();
        };
    });
</script>

<Modal {isOpen} {...rest} id={wrapperId} on:open={() => onopen?.()}>
    {@render children?.()}
</Modal>
