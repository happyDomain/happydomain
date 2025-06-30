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
    export const controls = {
        Open(domain: string): void { },
    };
</script>

<script lang="ts">
    import { run } from 'svelte/legacy';

    import { createEventDispatcher } from "svelte";

    // @ts-ignore
    import { escape } from "html-escaper";

    import {
        Button,
        Icon,
        Input,
        InputGroup,
        InputGroupText,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { addZoneService } from "$lib/api/zone";
    import DomainInput from "$lib/components/forms/resources/DomainInput.svelte";
    import { fqdn, validateDomain } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { Zone } from "$lib/model/zone";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    const toggle = () => (isOpen = !isOpen);


    interface Props {
        isOpen?: boolean;
        dn?: string;
        origin: Domain;
        value?: string;
    }

    let {
        isOpen = $bindable(false),
        dn = $bindable(""),
        origin,
        value = $bindable("")
    }: Props = $props();
    let zone = $thisZone;

    let validDomain: boolean | undefined = $state(undefined);
    let validSubDomain: boolean = $derived(validDomain && validateNewSubdomain(value));

    function validateNewSubdomain(value: string): boolean {
        if (!zone) return false;

        // Check domain doesn't already exists
        if (zone.services[value]) {
            return false;
        } else if (
            value.length > origin.domain.length &&
            value.indexOf(origin.domain) == value.length - origin.domain.length &&
            zone.services[value.substring(0, value.length - origin.domain.length)]
        ) {
            return false;
        } else if (
            value.length > origin.domain.length &&
            value.indexOf(origin.domain.substring(0, origin.domain.length - 1)) ==
                value.length - origin.domain.length + 1 &&
            zone.services[value.substring(0, value.length - origin.domain.length)]
        ) {
            return false;
        }

        return true;
    }

    let addAliasInProgress = $state(false);
    function submitAliasForm(e: FormDataEvent) {
        e.preventDefault();

        if (zone && validSubDomain) {
            addAliasInProgress = true;
            addZoneService(origin, zone.id, {
                _domain: value,
                _svctype: "svcs.CNAME",
                Service: { cname: { Target: dn ? dn : "@" } },
            }).then(
                (z) => {
                    thisZone.set(z);
                    addAliasInProgress = false;
                    toggle();
                },
                (err) => {
                    addAliasInProgress = false;
                    throw err;
                },
            );
        }
    }

    function Open(domain: string): void {
        dn = domain;
        value = "";
        isOpen = true;
    }

    controls.Open = Open;
</script>

<Modal {isOpen} {toggle}>
    <ModalHeader {toggle}>
        {$t("domains.add-an-alias", {domain: origin.domain})}
    </ModalHeader>
    <ModalBody>
        <form id="addAliasForm" onsubmit={submitAliasForm}>
            <p>
                {@html $t("domains.alias-creation", {
                    domain: `<span class="font-monospace">${escape(fqdn(dn, origin.domain))}</span>`,
                })}
            </p>
            <DomainInput
                autofocus
                {origin}
                bind:value
                on:validity-changed={(e) => validDomain = e.detail}
            />
            {#if validSubDomain}
                <div class="mt-3 text-center">
                    {$t("domains.alias-creation-sample")}<br />
                    <span class="font-monospace text-no-wrap">{fqdn(value, origin.domain)}</span>
                    <Icon class="mx-1" name="arrow-right" />
                    <span class="font-monospace text-no-wrap">{fqdn(dn, origin.domain)}</span>
                </div>
            {/if}
        </form>
    </ModalBody>
    <ModalFooter>
        <Button color="secondary" outline on:click={toggle}>
            {$t("common.cancel")}
        </Button>
        <Button
            type="submit"
            disabled={validSubDomain !== true || addAliasInProgress}
            form="addAliasForm"
            color="primary"
        >
            {#if addAliasInProgress}
                <Spinner size="sm" />
            {/if}
            {$t("domains.add-alias")}
        </Button>
    </ModalFooter>
</Modal>
