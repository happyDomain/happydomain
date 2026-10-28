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
    import { Spinner } from "@sveltestrap/sveltestrap";

    import RecordModal from "$lib/components/modals/Record.svelte";
    import TileLoader from "$lib/components/TileLoader.svelte";
    import type { Domain } from "$lib/model/domain";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        // The zone the pages below show, once loaded in thisZone.
        zoneId: string;
        children?: Snippet;
    }

    let { domain, zoneId, children }: Props = $props();
</script>

{#if $thisZone && $thisZone.id == zoneId}
    {@render children?.()}

    <RecordModal origin={domain} zone={$thisZone} />
{:else}
    <div class="flex-fill d-flex flex-column">
        <h2 class="d-flex align-items-center">
            <Spinner type="grow" />
            <span class="ms-2 mt-1 font-monospace">
                {domain.domain}
            </span>
        </h2>

        <div class="mt-4 text-center flex-fill">
            <TileLoader class="mb-2" />
            <p>{$t("wait.loading")}</p>
        </div>
    </div>
{/if}
