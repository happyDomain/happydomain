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
 import {
     Button,
     Input,
     InputGroup,
     Modal,
     ModalBody,
     ModalHeader,
     Spinner,
 } from 'sveltestrap';

 import ZoneList from '$lib/components/ZoneList.svelte';
 import { updateDomain } from '$lib/api/domains';
 import type { Domain } from '$lib/model/domain';
 import { groups, domains, refreshDomains } from '$lib/stores/domains';
 import { t } from '$lib/translations';

 export let isOpen = false;
 const toggle = () => (isOpen = !isOpen);

 let mygroups: Array<string> = [];
 $: if (!isOpen) mygroups = [];
 $: if (!mygroups.length) mygroups = $groups.map((s) => s).filter((s) => s != "" && s != 'undefined');

 let newgroup = "";
 function addGroup() {
     if (newgroup.length && mygroups.indexOf(newgroup) < 0) {
         mygroups.push(newgroup);
         mygroups = mygroups;
     }
     newgroup = "";
 }

 async function changeGroup(event: Event, domain: Domain) {
     if (event.currentTarget && event.currentTarget instanceof HTMLSelectElement) {
         domain.group = event.currentTarget.value;
         domain = await updateDomain(domain);
         refreshDomains();
     }
 }
</script>

<Modal
    {isOpen}
    {toggle}
    scrollable
    size="lg"
>
    <ModalHeader {toggle}>
        {$t('domaingroups.manage')}
    </ModalHeader>
    <ModalBody>
        {#if $domains == null}
            <div class="d-flex justify-content-center">
                <Spinner color="primary" />
            </div>
        {:else}
            <form on:submit|preventDefault={addGroup} class="mb-4">
                <InputGroup>
                    <Input
                        id="newgroup"
                        placeholder={$t('domaingroups.new')}
                        required
                        bind:value={newgroup}
                    />
                    <Button
                        type="submit"
                        color="primary"
                        disabled={newgroup.length < 1 && mygroups.indexOf(newgroup) >= 0}
                    >
                        {$t('common.add')}
                    </Button>
                </InputGroup>
            </form>
            <ZoneList
                class="mt-3"
                domains={$domains}
            >
                <div slot="badges" let:item={domain}>
                    <Input
                        type="select"
                        value={domain.group}
                        on:change={(event) => changeGroup(event, domain)}
                    >
                        <option value="">{$t('domaingroups.no-group')}</option>
                        {#each mygroups as group}
                            <option value={group}>{group}</option>
                        {/each}
                    </Input>
                </div>
            </ZoneList>
        {/if}
    </ModalBody>
</Modal>
