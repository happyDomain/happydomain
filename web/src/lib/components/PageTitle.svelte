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
        /** Main section title (e.g. "Zone Editor", "History") */
        title: string;
        /** Domain name displayed in monospace above the title */
        domain?: string;
        /** Optional subtitle or contextual hint below the title */
        subtitle?: string;
        /** Optional slot for status badges, health checks, or summary indicators */
        children?: Snippet;
    }

    let { title, domain, subtitle, children }: Props = $props();
</script>

<header class="page-title mb-4">
    <div class="d-flex align-items-end justify-content-between gap-3 flex-wrap pb-1">
        <div class="page-title-text">
            {#if domain}
                <div class="page-title-domain font-monospace">{domain}</div>
            {/if}
            <h1 class="display-5 page-title-heading">{title}</h1>
            {#if subtitle}
                <p class="page-title-subtitle text-muted">{subtitle}</p>
            {/if}
        </div>
        {#if children}
            <div class="page-title-summary">
                {@render children()}
            </div>
        {/if}
    </div>
</header>

<style>
    .page-title {
        padding: 0.75rem 0 1rem 0;
        border-bottom: 1px solid rgba(0, 0, 0, 0.07);
    }

    .page-title-text::before {
        content: "";
        display: block;
        width: 2rem;
        height: 2px;
        background: var(--bs-primary);
        border-radius: 1px;
        margin-bottom: 0.25rem;
    }

    .page-title-domain {
        font-size: 0.9rem;
        font-weight: 600;
        color: var(--bs-primary);
        letter-spacing: 0.06em;
        text-transform: lowercase;
        margin-top: 0.5rem;
        margin-bottom: 0.2rem;
    }

    .page-title-heading {
        margin: 0;
        color: #1a1a2e;
    }

    .page-title-subtitle {
        font-size: 1.1rem;
        margin-top: 0.2rem;
        margin-bottom: 0;
        line-height: 1.4;
    }

    .page-title-summary {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 0.4rem;
        padding-top: 0.25rem;
        flex-shrink: 0;
    }
</style>
