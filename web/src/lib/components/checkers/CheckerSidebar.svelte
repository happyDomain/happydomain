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
    import { Icon, Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { checkers } from "$lib/stores/checkers";

    let {
        currentCheckId,
    }: {
        currentCheckId: string;
    } = $props();
</script>

<div class="d-flex flex-column h-100">
    <a
        href="/checkers"
        class="sidebar-back d-flex align-items-center gap-1 mb-3 text-muted text-decoration-none fw-semibold"
    >
        <Icon name="chevron-left" />
        {$t("checkers.title")}
    </a>

    {#if $checkers}
        <ul class="list-unstyled mb-0 flex-fill overflow-auto">
            {#each Object.entries($checkers) as [checkerId, checkerInfo]}
                <li>
                    <a
                        href="/checkers/{encodeURIComponent(checkerId)}"
                        class="checker-item d-flex align-items-center gap-2 py-2 px-2 rounded text-decoration-none"
                        class:active={checkerId === currentCheckId}
                    >
                        <span class="text-truncate">
                            {checkerInfo.name || checkerId}
                        </span>
                    </a>
                </li>
            {/each}
        </ul>
    {:else}
        <div class="d-flex gap-2 align-items-center justify-content-center my-3 text-muted">
            <Spinner size="sm" color="primary" />
        </div>
    {/if}
</div>

<style>
    .checker-item {
        transition: background-color 0.15s;
    }

    .checker-item:hover {
        background-color: rgba(0, 0, 0, 0.06);
    }

    .checker-item.active {
        background-color: rgba(var(--bs-primary-rgb), 0.1);
    }
</style>
