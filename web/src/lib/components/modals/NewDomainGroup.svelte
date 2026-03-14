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

<script module lang="ts">
    import type { ModalController } from "$lib/model/modal_controller";

    export const controls: ModalController = {
        Open(): void {},
    };
</script>

<script lang="ts">
    import {
        Button,
        Input,
        InputGroup,
        Modal,
        ModalBody,
        ModalHeader,
    } from "@sveltestrap/sveltestrap";

    import { newlyGroups } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    interface Props {
        isOpen?: boolean;
    }

    let { isOpen = $bindable(false) }: Props = $props();
    const toggle = () => (isOpen = !isOpen);

    let newgroup = $state("");
    let inputEl: HTMLInputElement | undefined = $state();

    function focusInput() {
        inputEl?.focus();
    }

    function addGroup(e: SubmitEvent) {
        e.preventDefault();
        if (newgroup.length) {
            newlyGroups.update((gs) => (gs.includes(newgroup) ? gs : [...gs, newgroup]));
        }
        newgroup = "";
        isOpen = false;
    }

    function Open(): void {
        newgroup = "";
        isOpen = true;
    }

    controls.Open = Open;
</script>

<Modal {isOpen} {toggle} on:open={focusInput}>
    <ModalHeader {toggle}>
        {$t("domaingroups.new")}
    </ModalHeader>
    <ModalBody>
        <form onsubmit={addGroup}>
            <InputGroup>
                <Input
                    placeholder={$t("domaingroups.new")}
                    required
                    bind:value={newgroup}
                    bind:inner={inputEl}
                />
                <Button type="submit" color="primary" disabled={newgroup.length < 1}>
                    {$t("common.add")}
                </Button>
            </InputGroup>
        </form>
    </ModalBody>
</Modal>
