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
        Open(record: dnsRR, service: ServiceCombined) { },
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

    import { addServiceRecord, deleteServiceRecord, updateServiceRecord } from "$lib/api/service";
    import { fqdn, nsclass, nsrrtype } from "$lib/dns";
    import { rdatafields } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import type { Zone } from "$lib/model/zone";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    const toggle = () => (isOpen = !isOpen);

    interface Props {
        isOpen?: boolean;
        origin: Domain;
        zone: Zone;
    }

    let { isOpen = $bindable(false), origin, zone }: Props = $props();

    let service: ServiceCombined | undefined = $state(undefined);
    let record: dnsRR | undefined = $state(undefined);

    let addRecordInProgress = $state(false);
    let deleteRecordInProgress = $state(false);

    function deleteRecord() {
        deleteRecordInProgress = true;
        deleteServiceRecord(origin, zone.id, record).then(
            (z: Zone) => {
                dispatch("update-zone-records", z);
                deleteRecordInProgress = false;
                toggle();
            },
            (err: Error) => {
                deleteRecordInProgress = false;
                throw err;
            },
        );
    }

    function submitRecordForm(e: FormDataEvent) {
        e.preventDefault();

        addRecordInProgress = true;

        let action = addServiceRecord;
        if (record._id) {
            action = updateServiceRecord;
        }

        action(origin, zone.id, record).then(
            (z: Zone) => {
                dispatch("update-zone-records", z);
                addRecordInProgress = false;
                toggle();
            },
            (err: Error) => {
                addRecordInProgress = false;
                throw err;
            },
        );
    }

    function Open(r: dnsRR, s: ServiceCombined): void {
        record = r;
        service = s;
        isOpen = true;
    }

    controls.Open = Open;
</script>

{#if record}
    <Modal {isOpen} scrollable size="lg" {toggle}>
        <ModalHeader {toggle}>
            {#if record.Hdr.Class}
                {$t("records.update")}
            {:else if service}
                {@html $t("records.form-new", {
                    domain: `<span class="font-monospace">${escape(fqdn(service._domain, origin.domain))}</span>`,
                })}
            {/if}
        </ModalHeader>
        <ModalBody>
            <form id="addRRForm" onsubmit={submitRecordForm}>
                <Row>
                    <label
                        for="rr-hdr-name"
                        class="col-md-4 col-form-label text-truncate text-md-right text-primary"
                    >
                        {$t("domains.subdomain")[0].toUpperCase()}{$t(
                            "domains.subdomain",
                        ).substring(1)}
                    </label>
                    <Col md="8" class="d-flex flex-column">
                        <div class="flex-fill d-flex align-items-center">
                            <InputGroup size="sm">
                                <Input
                                    id="rr-hdr-name"
                                    type="text"
                                    class="fw-bold"
                                    bind:value={record.Hdr.Name}
                                />
                                <InputGroupText
                                    >.{fqdn(service._domain, origin.domain)}</InputGroupText
                                >
                            </InputGroup>
                        </div>
                    </Col>
                </Row>
                <Row>
                    <label
                        for="rr-hdr-class"
                        class="col-md-4 col-form-label text-truncate text-md-right text-primary"
                    >
                        {$t("records.class")[0].toUpperCase()}{$t("records.class").substring(1)}
                    </label>
                    <Col md="8" class="d-flex flex-column">
                        <div class="flex-fill d-flex align-items-center">
                            <Input
                                id="rr-hdr-class"
                                size="sm"
                                type="select"
                                required
                                bind:value={record.Hdr.Class}
                            >
                                {#each [1, 3, 4, 254] as i}
                                    <option value={i}>{nsclass(i)}</option>
                                {/each}
                            </Input>
                        </div>
                    </Col>
                </Row>
                <Row>
                    <label
                        for="rr-hdr-rrtype"
                        class="col-md-4 col-form-label text-truncate text-md-right text-primary"
                    >
                        {$t("records.rrtype")[0].toUpperCase()}{$t("records.rrtype").substring(1)}
                    </label>
                    <Col md="8" class="d-flex flex-column">
                        <div class="flex-fill d-flex align-items-center">
                            <Input
                                id="rr-hdr-rrtype"
                                type="select"
                                size="sm"
                                required
                                bind:value={record.Hdr.Rrtype}
                            >
                                {#each Array(260)
                                    .fill()
                                    .map((element, index) => index + 1) as i}
                                    {#if nsrrtype(i) != "#"}
                                        <option value={i}>{nsrrtype(i)}</option>
                                    {/if}
                                {/each}
                            </Input>
                        </div>
                    </Col>
                </Row>
                <Row>
                    <label
                        for="rr-hdr-ttl"
                        class="col-md-4 col-form-label text-truncate text-md-right text-primary"
                    >
                        {$t("records.ttl")[0].toUpperCase()}{$t("records.ttl").substring(1)}
                    </label>
                    <Col md="8" class="d-flex flex-column">
                        <div class="flex-fill d-flex align-items-center">
                            <InputGroup size="sm">
                                <Input
                                    id="rr-hdr-ttl"
                                    type="number"
                                    required
                                    bind:value={record.Hdr.Ttl}
                                />
                                <InputGroupText>s</InputGroupText>
                            </InputGroup>
                        </div>
                    </Col>
                </Row>
                <hr />
                {#if record.Hdr.Rrtype && rdatafields(record.Hdr.Rrtype).length > 0}
                  {#each rdatafields(record.Hdr.Rrtype) as k}
                    {#if k != "Hdr"}
                        {@const v = record[k]}
                        <Row>
                            <label
                                for="rr-{k}"
                                class="col-md-4 col-form-label text-truncate text-md-right text-primary"
                            >
                                {k}
                            </label>
                            <Col md="8" class="d-flex flex-column">
                                <div class="flex-fill d-flex align-items-center">
                                    <Input id="rr-{k}" size="sm" type="text" bind:value={record[k]} />
                                </div>
                            </Col>
                        </Row>
                    {/if}
                  {/each}
                {/if}
            </form>
        </ModalBody>
        <ModalFooter>
            <div class="ms-auto"></div>
            {#if record.Hdr.Class}
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
            {#if record.Hdr.Class}
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
