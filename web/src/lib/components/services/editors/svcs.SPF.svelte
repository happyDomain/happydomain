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
    import { tick } from "svelte";

    import { Button, Icon, InputGroup, ListGroup, ListGroupItem } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
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

    let inputRefs: HTMLInputElement[] = [];

    async function addDirective() {
        let newIdx: number;
        if (val.f.length >= 1 && val.f[val.f.length - 1].indexOf("all") >= 0) {
            newIdx = val.f.length - 1;
            val.f.splice(val.f.length - 1, 0, "");
        } else {
            newIdx = val.f.length;
            val.f.push("");
        }
        await tick();
        inputRefs[newIdx]?.focus();
    }

    function delDirective(idx: number) {
        val.f.splice(idx, 1);
    }
</script>

{#if $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Sender Policy Framework</h4>

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
            <ListGroupItem class="p-0">
                <InputGroup>
                    <input
                        class="form-control border-0"
                        bind:value={val.f[i]}
                        bind:this={inputRefs[i]}
                    />
                    <Button
                        type="button"
                        color="link"
                        class="text-danger"
                        onclick={() => delDirective(i)}
                    >
                        <Icon name="trash" />
                    </Button>
                </InputGroup>
            </ListGroupItem>
        {/each}
        <ListGroupItem tag="button" class="text-muted fst-italic" action onclick={addDirective}>
            New directive
        </ListGroupItem>
    </ListGroup>
</div>
