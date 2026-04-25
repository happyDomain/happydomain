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
    import type { dnsResource, dnsTypeTXT } from "$lib/dns_rr";
    import { newRR } from "$lib/dns_rr";
    import { parseSPF, stringifySPF } from "$lib/services/spf";

    interface Props {
        dn: string;
        origin: Domain;
        value: dnsResource;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    const DEFAULT_SPF = "v=spf1 -all";
    const ALL_RE = /^[-~?+]?all$/;

    if (!value["txt"]) {
        const txtRecord = newRR(dn, 16) as dnsTypeTXT; // TXT record type is 16
        txtRecord.Txt = DEFAULT_SPF;
        value["txt"] = txtRecord;
    }

    const initial = parseSPF(value["txt"].Txt || DEFAULT_SPF);
    let v = $state(initial.v ?? "spf1");
    let f = $state(initial.f);

    $effect(() => {
        value["txt"]!.Txt = stringifySPF({ v, f });
    });

    const type = "svcs.SPF";

    let inputRefs: HTMLInputElement[] = $state([]);

    async function addDirective() {
        let newIdx: number;
        if (f.length >= 1 && ALL_RE.test(f[f.length - 1])) {
            newIdx = f.length - 1;
            f.splice(newIdx, 0, "");
        } else {
            newIdx = f.length;
            f.push("");
        }
        await tick();
        inputRefs[newIdx]?.focus();
    }

    function delDirective(idx: number) {
        f.splice(idx, 1);
    }
</script>

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
        bind:value={v}
    />

    <h5 class="text-primary pb-1 border-bottom border-1">Directives</h5>
    <ListGroup>
        {#each f as _, i (i)}
            <ListGroupItem class="p-0">
                <InputGroup>
                    <input
                        class="form-control border-0"
                        bind:value={f[i]}
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
