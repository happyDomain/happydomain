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

<script lang="ts" generics="T">
    import type { Snippet } from "svelte";
    import {
        Button,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { toasts } from "$lib/stores/toasts";

    interface Props {
        title: string;
        icon: string;
        color?: string;
        label: string;
        confirmText: string;
        run: () => Promise<{ data?: T; error?: unknown }>;
        disabled?: boolean;
        disabledReason?: string;
        onDone?: (data: T) => void;
        children?: Snippet;
        result?: Snippet<[T]>;
        class?: string;
    }

    let {
        title,
        icon,
        color = "secondary",
        label,
        confirmText,
        run,
        disabled = false,
        disabledReason,
        onDone,
        children,
        result,
        class: className = "",
    }: Props = $props();

    let isProcessing = $state(false);
    let lastResult = $state<{ data: T } | null>(null);

    async function execute() {
        if (!confirm(confirmText)) return;

        isProcessing = true;
        try {
            const { data, error: err } = await run();
            if (err) throw new Error((err as { errmsg?: string })?.errmsg || String(err));

            lastResult = { data: data as T };
            toasts.addToast({
                type: "success",
                title: `${title}: done`,
                timeout: 5000,
            });
            onDone?.(data as T);
        } catch (err) {
            toasts.addErrorToast({
                title: `${title} failed`,
                message: err instanceof Error ? err.message : "Unknown error occurred",
                timeout: 10000,
            });
        } finally {
            isProcessing = false;
        }
    }
</script>

<Card class="h-100 {className}">
    <CardHeader>
        <Icon name={icon}></Icon>
        {title}
    </CardHeader>
    <CardBody class="d-flex flex-column">
        <div class="text-muted mb-3 flex-fill">
            {@render children?.()}
        </div>
        <div>
            <Button {color} disabled={disabled || isProcessing} onclick={execute}>
                {#if isProcessing}
                    <Spinner size="sm" class="me-1" /> Processing...
                {:else}
                    <Icon name={icon} class="me-1"></Icon> {label}
                {/if}
            </Button>
            {#if disabled && disabledReason}
                <small class="d-block text-muted mt-2">{disabledReason}</small>
            {/if}
        </div>
    </CardBody>
    {#if lastResult && result}
        <CardFooter class="p-0">
            {@render result(lastResult.data)}
        </CardFooter>
    {/if}
</Card>
