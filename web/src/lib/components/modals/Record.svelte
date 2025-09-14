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
    import type { dnsRR } from "$lib/dns_rr";

    export const controls = {
        Open(record: dnsRR | null, dn: string) { },
    };
</script>

<script lang="ts">
    import { createEventDispatcher } from "svelte";

    import {
        Button,
        Col,
        Icon,
        Input,
        InputGroup,
        InputGroupText,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { addZoneRecord, deleteZoneRecord, updateZoneRecord } from "$lib/api/zone";
    import RecordEditor from "$lib/components/records/Editor.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { Zone } from "$lib/model/zone";
    import { thisZone } from "$lib/stores/thiszone";
    import { emptyRR } from "$lib/dns";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    const toggle = () => (isOpen = !isOpen);

    interface Props {
        isOpen?: boolean;
        origin: Domain;
        zone: Zone;
    }

    let { isOpen = $bindable(false), origin, zone }: Props = $props();

    let dn: string = $state("");
    let record: dnsRR | undefined = $state(undefined);
    let initialrecord: dnsRR | undefined = $state(undefined);

    let addRecordInProgress = $state(false);
    let deleteRecordInProgress = $state(false);

    function deleteRecord() {
        if (!record) return;
        deleteRecordInProgress = true;
        deleteZoneRecord(origin, zone.id, dn, record).then(
            (z: Zone) => {
                thisZone.set(z);
                deleteRecordInProgress = false;
                toggle();
            },
            (err: Error) => {
                deleteRecordInProgress = false;
                throw err;
            },
        );
    }

    function submitRecordForm(e: SubmitEvent) {
        e.preventDefault();

        if (!record) return;

        addRecordInProgress = true;

        let promise: Promise<Zone>;
        if (initialrecord) {
            promise = updateZoneRecord(origin, zone.id, dn, record, initialrecord);
        } else {
            promise = addZoneRecord(origin, zone.id, dn, record);
        }

        promise.then(
            (z: Zone) => {
                thisZone.set(z);
                addRecordInProgress = false;
                toggle();
            },
            (err: Error) => {
                addRecordInProgress = false;
                throw err;
            },
        );
    }

    function Open(r: dnsRR, d: string): void {
        if (r) {
            initialrecord = JSON.parse(JSON.stringify(r));
            record = r;
        } else {
            initialrecord = undefined;
            record = emptyRR();
            record.Hdr.Rrtype = 1;
        }
        dn = d;
        isOpen = true;
    }

    controls.Open = Open;
</script>

{#if record}
    <Modal {isOpen} scrollable size="lg" {toggle}>
        <ModalHeader {toggle}>
            {#if initialrecord}
                {$t("records.update")}
            {:else}
                {@html $t("records.form-new", {
                    domain: `<span class="font-monospace">${escape(origin.domain)}</span>`,
                })}
            {/if}
        </ModalHeader>
        <ModalBody>
            <form id="addRRForm" onsubmit={submitRecordForm}>
                <RecordEditor
                    bind:dn={dn}
                    {origin}
                    bind:record={record}
                />
            </form>
        </ModalBody>
        <ModalFooter>
            <div class="ms-auto"></div>
            {#if initialrecord}
                <Button
                    color="danger"
                    disabled={addRecordInProgress ||
                        deleteRecordInProgress ||
                        record.Hdr.Rrtype == 6}
                    title={$t("records.delete")}
                    on:click={deleteRecord}
                >
                    {#if deleteRecordInProgress}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="trash" />
                    {/if}
                </Button>
            {/if}
            <Button color="secondary" on:click={toggle}>
                {$t("common.cancel")}
            </Button>
            {#if initialrecord}
                <Button
                    disabled={addRecordInProgress || deleteRecordInProgress}
                    form="addRRForm"
                    type="submit"
                    color="success"
                >
                    {#if addRecordInProgress}
                        <Spinner size="sm" />
                    {/if}
                    {$t("records.update")}
                </Button>
            {:else}
                <Button form="addRRForm" type="submit" color="primary">
                    {$t("records.add")}
                </Button>
            {/if}
        </ModalFooter>
    </Modal>
{/if}
