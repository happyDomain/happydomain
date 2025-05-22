<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    import { Input } from "@sveltestrap/sveltestrap";

    import { domains_by_groups } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    interface Props {
        class?: string;
        selectedDomain: string;
    }

    let { class: className = "", selectedDomain = $bindable() }: Props = $props();
</script>

{#key $domains_by_groups}
    <Input type="select" class={className} bind:value={selectedDomain}>
        {#each Object.keys($domains_by_groups) as gname}
            {@const group = $domains_by_groups[gname]}
            <optgroup
                label={gname == "undefined" || !gname
                      ? $t("domaingroups.no-group")
                      : gname}
            >
                {#each group as domain}
                    <option value={domain.id}>{domain.domain}</option>
                {/each}
            </optgroup>
        {/each}
    </Input>
{/key}
