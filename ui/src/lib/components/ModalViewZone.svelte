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

<script context="module" lang="ts">
 import type { ModalController } from '$lib/model/modal_controller';

 export const controls: ModalController = { };
</script>

<script lang="ts">
 import {
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import {
     viewZone as APIViewZone,
 } from '$lib/api/zone';
 import type { Domain } from '$lib/model/domain';
 import { t } from '$lib/translations';

 export let isOpen = false;

 let zoneContent: null | string = null;
 function Open(domain: Domain, selectedHistory: string): void {
     zoneContent = null;
     isOpen = true;
     APIViewZone(domain, selectedHistory).then(
         (v: string) => zoneContent = v,
         (err: any) => {
             isOpen = false;
             throw err;
         }
     );
 }

 function toggle(): void {
     isOpen = !isOpen;
 }

 controls.Open = Open;
</script>

<Modal
    isOpen={isOpen}
    size="lg"
    scrollable
    {toggle}
>
    <ModalHeader {toggle}>{$t('domains.view.title')}</ModalHeader>
    <ModalBody>
        {#if zoneContent}
            <pre style="overflow: initial">{zoneContent}</pre>
        {:else}
            <div class="my-2 text-center">
                <Spinner label="Spinning" />
                <p>{$t('wait.formating')}</p>
            </div>
        {/if}
    </ModalBody>
</Modal>
