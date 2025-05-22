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
    import type { Correction } from "$lib/model/correction";
    import { t } from "$lib/translations";

    interface Props {
        zoneDiff: Array<Correction>;
    }

    let { zoneDiff }: Props = $props();

    let zoneDiffCreated = $derived(zoneDiff.filter((c) => c.kind == 1).length);
    let zoneDiffModified = $derived(zoneDiff.filter((c) => c.kind == 2).length);
    let zoneDiffDeleted = $derived(zoneDiff.filter((c) => c.kind == 3).length);
    let zoneDiffOther = $derived(zoneDiff.filter((c) => c.kind == 99).length);
</script>

{#if zoneDiff && zoneDiff.length}
    {#if zoneDiffCreated}
        <span class="text-success">
            {$t("domains.apply.additions", { count: zoneDiffCreated })}
        </span>
    {/if}
    {#if zoneDiffCreated && zoneDiffDeleted}
        &ndash;
    {/if}
    {#if zoneDiffDeleted}
        <span class="text-danger">
            {$t("domains.apply.deletions", { count: zoneDiffDeleted })}
        </span>
    {/if}
    {#if (zoneDiffCreated || zoneDiffDeleted) && zoneDiffModified}
        &ndash;
    {/if}
    {#if zoneDiffModified}
        <span class="text-warning">
            {$t("domains.apply.modifications", { count: zoneDiffModified })}
        </span>
    {/if}
    {#if (zoneDiffCreated || zoneDiffDeleted || zoneDiffModified) && zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0}
        &ndash;
    {/if}
    {#if zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0}
        <span class="text-info">
            {$t("domains.apply.others", {
                count: zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified,
            })}
        </span>
    {/if}
{:else}
    {$t("domains.apply.modifications", { count: 0 })}
{/if}
