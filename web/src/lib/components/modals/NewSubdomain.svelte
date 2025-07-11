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

<script module lang="ts">
    import type { ModalController } from "$lib/model/modal_controller";

    export const controls: ModalController = {
        Open(): void { },
    };
</script>

<script lang="ts">
    import { createEventDispatcher } from "svelte";

    // @ts-ignore
    import { escape } from "html-escaper";
    import { Modal, ModalBody } from "@sveltestrap/sveltestrap";

    import DomainInput from "$lib/components/inputs/Domain.svelte";
    import ModalFooter from "$lib/components/modals/Footer.svelte";
    import ModalHeader from "$lib/components/modals/Header.svelte";
    import { validateDomain } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    const toggle = () => (isOpen = !isOpen);

    interface Props {
        isOpen?: boolean;
        origin: Domain;
        value?: string;
    }

    let { isOpen = $bindable(false), origin, value = $bindable("") }: Props = $props();

    let validDomain: boolean | undefined = $state(undefined);

    function submitSubdomainForm(e: SubmitEvent) {
        e.preventDefault();

        if (validDomain) {
            toggle();
            dispatch("show-next-modal", value);
        }
    }

    function Open(): void {
        isOpen = true;
        value = "";
    }

    controls.Open = Open;
</script>

<Modal {isOpen} {toggle}>
    <ModalHeader {toggle} dn={origin.domain} />
    <ModalBody>
        <form id="addSubdomainForm" onsubmit={submitSubdomainForm}>
            <p>
                {@html $t("domains.form-new-subdomain", {
                    domain: `<span class="font-monospace">${escape(origin.domain)}</span>`,
                })}
            </p>

            <DomainInput
                {origin}
                bind:value
                on:validity-changed={(e) => validDomain = e.detail}
            />
        </form>
    </ModalBody>
    <ModalFooter canContinue={validDomain === true} form="addSubdomainForm" step={0} {toggle} />
</Modal>
