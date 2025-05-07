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
    import type { ModalController } from "$lib/model/modal_controller";

    export const controls: ModalController = {};
</script>

<script lang="ts">
    import { createEventDispatcher } from "svelte";

    // @ts-ignore
    import { escape } from "html-escaper";
    import { Input, InputGroup, InputGroupText, Modal, ModalBody } from "@sveltestrap/sveltestrap";

    import ModalFooter from "$lib/components/domains/ModalFooter.svelte";
    import ModalHeader from "$lib/components/domains/ModalHeader.svelte";
    import { validateDomain } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    export let isOpen = false;
    const toggle = () => (isOpen = !isOpen);

    export let origin: Domain;
    export let value: string = "";

    let newDomainState: boolean | undefined = undefined;
    $: newDomainState = value ? validateNewSubdomain(value) : undefined;

    let endsWithOrigin = false;
    $: endsWithOrigin =
        value.endsWith(origin.domain) ||
        value.endsWith(origin.domain.substring(0, origin.domain.length - 1));

    let newDomainAppend: string | null = null;
    $: {
        if (endsWithOrigin) {
            newDomainAppend = null;
        } else if (value.length > 0) {
            newDomainAppend = "." + origin.domain;
        } else {
            newDomainAppend = origin.domain;
        }
    }

    function validateNewSubdomain(value: string): boolean | undefined {
        newDomainState = validateDomain(value, origin.domain);
        return newDomainState;
    }

    function submitSubdomainForm() {
        if (validateNewSubdomain(value)) {
            toggle();
            dispatch("show-next-modal", value);
        }
    }

    function Open(domain): void {
        isOpen = true;
        value = "";
    }

    controls.Open = Open;
</script>

<Modal {isOpen} {toggle}>
    <ModalHeader {toggle} dn={origin.domain} />
    <ModalBody>
        <form id="addSubdomainForm" on:submit|preventDefault={submitSubdomainForm}>
            <p>
                {@html $t("domains.form-new-subdomain", {
                    domain: `<span class="font-monospace">${escape(origin.domain)}</span>`,
                })}

                <InputGroup>
                    <Input
                        autofocus
                        class="font-monospace"
                        placeholder={$t("domains.placeholder-new-sub")}
                        invalid={newDomainState === false}
                        valid={newDomainState === true}
                        bind:value
                    />
                    {#if newDomainAppend}
                        <InputGroupText class="font-monospace">
                            {newDomainAppend}
                        </InputGroupText>
                    {/if}
                </InputGroup>
            </p>
        </form>
    </ModalBody>
    <ModalFooter canContinue={newDomainState === true} form="addSubdomainForm" step={0} {toggle} />
</Modal>
