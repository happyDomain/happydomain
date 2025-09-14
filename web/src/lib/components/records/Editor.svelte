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
        Col,
        Input,
        InputGroup,
        InputGroupText,
        Row,
    } from "@sveltestrap/sveltestrap";

    import { fqdn, nsclass, nsrrtype } from "$lib/dns";
    import { rdatafields, type dnsRR } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        record: dnsRR;
    }

    let {
        dn = $bindable(""),
        origin,
        record = $bindable(),
    }: Props = $props();
</script>

<Row>
    <label
        for="rr-hdr-name"
        class="col-md-4 col-form-label text-truncate text-md-right text-primary"
    >
        {$t("domains.subdomain")[0].toUpperCase()}{$t("domains.subdomain").substring(1)}
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
                <InputGroupText>.{#if dn}{fqdn(dn, origin.domain)}{:else}{origin.domain}{/if}</InputGroupText>
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
                bsSize="sm"
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
                bsSize="sm"
                required
                bind:value={record.Hdr.Rrtype}
            >
            {#each Array.from({ length: 260 }, (_, index) => index + 1) as i}
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
            {@const v = (record as any)[k]}
            <Row>
                <label
                    for="rr-{k}"
                    class="col-md-4 col-form-label text-truncate text-md-right text-primary"
                >
                    {k}
                </label>
                <Col md="8" class="d-flex flex-column">
                    <div class="flex-fill d-flex align-items-center">
                        <Input id="rr-{k}" bsSize="sm" type="text" bind:value={(record as any)[k]} />
                    </div>
                </Col>
            </Row>
        {/if}
    {/each}
{/if}
