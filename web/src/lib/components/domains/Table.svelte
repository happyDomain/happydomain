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
    import type { ClassValue } from "svelte/elements";

    import { Spinner, Table } from "@sveltestrap/sveltestrap";

    import { deleteDomain } from "$lib/api/domains";
    import DomainTableRow from "$lib/components/domains/DomainTableRow.svelte";
    import type { HappydnsDomainWithCheckStatus } from "$lib/api-base/types.gen";
    import { refreshDomains } from "$lib/stores/domains";
    import { providersSpecs, refreshProvidersSpecs } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    interface Props {
        class?: ClassValue;
        items: Array<HappydnsDomainWithCheckStatus>;
        [key: string]: unknown;
    }

    let { class: className, items, ...rest }: Props = $props();

    if (!$providersSpecs) refreshProvidersSpecs();

    async function delDomain(event: Event, item: HappydnsDomainWithCheckStatus) {
        event.stopPropagation();

        if (!confirm($t("domains.alert.remove", { domain: item.domain }))) return;

        await deleteDomain(item.id);
        refreshDomains();
    }
</script>

{#if !items}
    <div class="d-flex gap-2 align-items-center justify-content-center my-3 {className}">
        <Spinner color="primary" />
        {$t("wait.retrieving-domains")}
    </div>
{:else if items.length === 0}
    <div class="text-center my-3 {className}">
        {$t("domains.filtered-no-result")}
    </div>
{:else}
    <Table class={className} striped hover responsive {...rest}>
        <thead>
            <tr>
                <th>{$t("common.domain")}</th>
                <th>{$t("domaingroups.title")}</th>
                <th>{$t("domains.view.provider")}</th>
                <th>Status</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            {#each items as item (item.id)}
                <DomainTableRow domain={item} ondelete={delDomain} />
            {/each}
        </tbody>
    </Table>
{/if}
