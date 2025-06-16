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
    import {
        Button,
        Card,
        Icon,
    } from "@sveltestrap/sveltestrap";

    import DomainGroupList from "$lib/components/domain-groups/DomainGroupList.svelte";
    import DomainGroupModal from "$lib/components/domain-groups/DomainGroupModal.svelte";
    import { domains } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    export { className as class };
    let className = "";

    let isGroupModalOpen = false;
    export let filteredGroup: string | null = null
</script>

{#if $domains && $domains.length}
    <Card class="mb-3 ${className}">
        <div class="card-header d-flex justify-content-between">
            {$t("domaingroups.title")}
            <Button
                type="button"
                size="sm"
                color="light"
                title={$t("domaingroups.manage")}
                on:click={() => (isGroupModalOpen = true)}
            >
                <Icon name="grid-fill" />
            </Button>
        </div>
        <DomainGroupList flush bind:selectedGroup={filteredGroup} />
    </Card>
{/if}

<DomainGroupModal bind:isOpen={isGroupModalOpen} />
