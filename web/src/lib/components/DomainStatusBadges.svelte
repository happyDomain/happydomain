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
    import { Badge } from "@sveltestrap/sveltestrap";

    import { tOr } from "$lib/translations";
    import { statusColor } from "$lib/utils/domaininfo";

    interface Props {
        status: string[];
    }

    let { status }: Props = $props();

    function description(code: string): string | undefined {
        return $tOr(`domaininfo.status-descriptions.${code}`, undefined);
    }
</script>

<div class="d-flex flex-wrap gap-2">
    {#each status as code, i (i)}
        <Badge color={statusColor(code)} title={description(code) ?? code}>
            {code}
        </Badge>
    {/each}
</div>
{#if status.length === 1 && description(status[0])}
    <p class="text-muted small mt-1 mb-0">{description(status[0])}</p>
{/if}
