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

    export let zoneDiff: Array<Correction>;

    let zoneDiffCreated = 0;
    let zoneDiffDeleted = 0;
    let zoneDiffModified = 0;
    let zoneDiffOther = 0;

    $: {
        zoneDiffCreated = 0;
        zoneDiffDeleted = 0;
        zoneDiffModified = 0;
        zoneDiffOther = 0;

        if (zoneDiff && zoneDiff.length) {
            zoneDiff.forEach((c: Correction) => {
                if (c.kind == 1) {
                    zoneDiffCreated += 1;
                } else if (c.kind == 2) {
                    zoneDiffModified += 1;
                } else if (c.kind == 3) {
                    zoneDiffDeleted += 1;
                } else if (c.kind == 99) {
                    zoneDiffOther += 1;
                }
            });
        }
    }
</script>

{#if zoneDiff}
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
