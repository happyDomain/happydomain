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

<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        class?: string;
        title: string;
        subtitle?: string;
        /** Buttons aligned on the right of the title */
        actions?: Snippet;
        children: Snippet;
    }

    let { class: className = "mt-9", title, subtitle, actions, children }: Props = $props();
</script>

<section class={className}>
    <div class="d-flex align-items-end justify-content-between gap-3 mb-4">
        <div>
            <h2 class="fs-5 fw-semibold text-body-emphasis mb-0">{title}</h2>
            {#if subtitle}
                <p class="small text-body-secondary mt-1 mb-0">{subtitle}</p>
            {/if}
        </div>
        {#if actions}
            {@render actions()}
        {/if}
    </div>
    {@render children()}
</section>
