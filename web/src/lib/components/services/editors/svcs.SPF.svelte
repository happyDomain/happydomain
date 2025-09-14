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
    import { Button, Icon, Input, InputGroup, ListGroup, ListGroupItem } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { servicesSpecs } from "$lib/stores/services";
    import type { dnsResource, dnsTypeTXT } from "$lib/dns_rr";
    import { newRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        value: dnsResource;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    function parseSPF(val: string) {
        const fields = val.split(" ");

        return {
            v: fields[0].replace(/^v=/, ""),
            f: fields.slice(1),
        };
    }
    function stringifySPF(val: any) {
        return "v=" + (val["v"] ? val["v"] : "spf1") + " " + val.f.join(" ");
    }
    let val = $state(parseSPF(value["txt"]?.Txt || "v=spf1 -all"));

    $effect(() => {
        if (value["txt"]?.Txt) {
            val = parseSPF(value["txt"].Txt);
        }
    });
    $effect(() => {
        if (!value["txt"]) {
            const txtRecord = newRR(dn, 16) as dnsTypeTXT; // TXT record type is 16
            txtRecord.Txt = "v=spf1 -all";
            value["txt"] = txtRecord;
        }
        if (value["txt"]) {
            value["txt"].Txt = stringifySPF(val);
        }
    });

    const type = "svcs.SPF";

    function addDirective() {
        if (val.f.length > 1 && val.f[val.f.length - 1].indexOf("all") >= 0) {
            val.f.splice(val.f.length-1, 0, "");
        } else {
            val.f.push("");
        }
    }

    function delDirective(idx: number) {
        val.f.splice(idx, 1);
    }
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Sender Policy Framework</h4>
    {#if value["txt"]}
        <RecordLine class="mb-4" {dn} {origin} bind:rr={value["txt"]} />
    {/if}

    <BasicInput
        edit
        index="v"
        specs={{
              id: "v",
              label: "Version",
              placeholder: "spf1",
              type: "string",
              description: "Defines the version of SPF to use.",
              }}
        bind:value={val.v}
    />

    <h5 class="text-primary pb-1 border-bottom border-1">Directives</h5>
    <ListGroup>
        {#each val.f as directive, i}
            <ListGroupItem>
                <InputGroup>
                    <Input
                        bsSize="sm"
                        bind:value={val.f[i]}
                    />
                    <Button
                        type="button"
                        color="danger"
                        outline
                        size="sm"
                        onclick={() => delDirective(i)}
                    >
                        <Icon name="trash" />
                    </Button>
                </InputGroup>
            </ListGroupItem>
        {/each}
        <ListGroupItem
            tag="button"
            class="text-muted fst-italic"
            action
            onclick={addDirective}
        >
            New directive
        </ListGroupItem>
    </ListGroup>
</div>
