<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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

<script lang="ts">
    import type { Snippet } from "svelte";

    import { t } from "$lib/translations";

    interface Props {
        toggle: () => void;
        // What the modal does, shown above the domain name.
        title: string;
        // The name worked on, relative to origin; empty for the origin itself.
        dn: string;
        origin: string;
        step?: number;
        steps?: number;
        children?: Snippet;
    }

    let { toggle, title, dn, origin, step, steps, children }: Props = $props();

    // The subdomain stands out, the origin it belongs to recedes.
    let sub = $derived(dn === "@" ? "" : dn);
</script>

<div class="modal-header d-block bg-light">
    <div class="d-flex align-items-start gap-3">
        <div class="flex-fill min-w-0 text-break">
            <div class="small fw-semibold text-uppercase text-body-secondary">{title}</div>
            <h5 class="modal-title font-monospace">
                {#if sub.endsWith(".")}
                    <span class="text-primary">{sub}</span>
                {:else if sub}
                    <span class="text-primary">{sub}</span><span class="text-muted fw-normal"
                        >.{origin}</span
                    >
                {:else}
                    {origin}
                {/if}
            </h5>
        </div>
        {#if step && steps}
            <div class="d-flex align-items-center gap-2 pt-1 small text-nowrap text-body-secondary">
                {$t("common.step-of", { step, count: steps })}
                <span class="d-flex gap-1" aria-hidden="true">
                    {#each { length: steps } as _, i (i)}
                        <span
                            class="d-inline-block p-1 rounded-circle {i < step
                                ? 'bg-primary'
                                : 'bg-secondary-subtle'}"
                        ></span>
                    {/each}
                </span>
            </div>
        {/if}
        <button type="button" class="btn-close" aria-label={$t("common.close")} onclick={toggle}
        ></button>
    </div>
    {#if children}
        <div class="mt-3">
            {@render children()}
        </div>
    {/if}
</div>
