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
    import { Button, Icon, Input } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { servicesSpecs } from "$lib/stores/services";
    import { parseKeyValueTxt } from "$lib/dns";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        value: any;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    function parseDKIM(val) {
        const kv = parseKeyValueTxt(val);

        kv.h = kv.h ? kv.h.split(":") : [];
        kv.s = kv.s ? kv.s.split(":") : [];
        kv.t = kv.t ? kv.s.split(":") : [];

        return kv;
    }
    function stringifyDKIM(val) {
        const sep = (value["txt"].Txt.indexOf("; ") >= 0 ? "; " : ";");

        return "v=" + (val["v"] ? val["v"] : "DMARCv1") +
                  (val["g"] ? sep + "g=" + val["g"] : "") +
                  (val["h"] ? sep + "h=" + val["h"].join(":") : "") +
                  (val["k"] ? sep + "k=" + val["k"] : "") +
                  (val["n"] ? sep + "n=" + val["n"] : "") +
                  (val["p"] ? sep + "p=" + val["p"] : "") +
                  (val["s"] ? sep + "s=" + val["s"].join(":") : "") +
                  (val["t"] ? sep + "t=" + val["t"].join(":") : "");
    }
    let val = $state(parseDKIM(value["txt"].Txt));
    $effect(() => {
        val = parseDKIM(value["txt"].Txt);
    });
    $effect(() => {
        value["txt"].Txt = stringifyDKIM(val);
    });

    let selector = $state(value["txt"].Hdr.Name.replace("._domainkey", ""));
    $effect(() => {
        selector = value["txt"].Hdr.Name.replace("._domainkey", "");
    });
    $effect(() => {
        value["txt"].Hdr.Name = selector + "._domainkey"
    });

    const type = "svcs.DKIM";

    function addDirective() {
        if (val.f.length > 1 && val.f[val.f.length - 1].indexOf("all") >= 0) {
            val.f.splice(val.f.length-1, 0, "");
        } else {
            val.f.push("");
        }
    }

    function delDirective(idx) {
        val.f.splice(idx, 1);
    }
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">DomainKeys Identified Mail</h4>

    <RecordLine class="mb-4" {dn} {origin} bind:rr={value["txt"]} />

    <form id="addSvcForm">
        <BasicInput
            edit
            specs={{
                  id: "v",
                  label: "Version",
                  placeholder: "DKIM1",
                  type: "string",
                  description: "Defines the version of DKIM to use.",
                  }}
            bind:value={val.v}
        />

        <BasicInput
            edit
            specs={{
                  id: "selector",
                  label: "Selector",
                  placeholder: "mail",
                  type: "string",
                  description: "Name of the key.",
                  }}
            bind:value={selector}
        />

        <h4 class="mt-1 text-primary pb-1 border-bottom border-1">Hash Algorithms</h4>
        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th>Hash Algorithms</th>
                </tr>
            </thead>
            <tbody>
                {#if val.h && val.h.length}
                    {#each val.h as rua, idx}
                        <tr>
                            <td>
                                <Input
                                    size="sm"
                                    bind:value={val.h[idx]}
                                />
                            </td>
                            <td>
                                <Button
                                    type="button"
                                    color="danger"
                                    outline
                                    size="sm"
                                    onclick={() => val.h.splice(idx, 1)}
                                >
                                    <Icon name="trash" />
                                </Button>
                            </td>
                        </tr>
                    {/each}
                {:else}
                    <tr>
                        <td
                            colspan={2}
                            class="fst-italic text-center"
                        >
                            {$t("common.no-content")}
                        </td>
                    </tr>
                {/if}
            </tbody>
            <tfoot>
                <tr>
                    <td colspan="1">
                        <Button
                            type="button"
                            color="primary"
                            outline
                            size="sm"
                            onclick={() => val.h.push("")}
                        >
                            <Icon name="plus" />
                            {$t("common.new-row")}
                        </Button>
                    </td>
                </tr>
            </tfoot>
        </table>

        <BasicInput
            edit
            specs={{
                  id: "k",
                  label: "Key Type",
                  choices: ["rsa", "ed25519"],
                  type: "string",
                  }}
            bind:value={val.k}
        />

        <BasicInput
            edit
            specs={{
                  id: "n",
                  label: "Notes",
                  type: "string",
                  description: "Notes intended for a foreign postmaster."
                  }}
            bind:value={val.n}
        />

        <BasicInput
            edit
            specs={{
                  id: "p",
                  label: "Public Key",
                  type: "string",
                  }}
            bind:value={val.p}
        />

        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th>Service Types</th>
                </tr>
            </thead>
            <tbody>
                {#if val.s && val.s.length}
                    {#each val.s as rua, idx}
                        <tr>
                            <td>
                                <Input
                                    size="sm"
                                    bind:value={val.s[idx]}
                                />
                            </td>
                            <td>
                                <Button
                                    type="button"
                                    color="danger"
                                    outline
                                    size="sm"
                                    onclick={() => val.s.splice(idx, 1)}
                                >
                                    <Icon name="trash" />
                                </Button>
                            </td>
                        </tr>
                    {/each}
                {:else}
                    <tr>
                        <td
                            colspan={2}
                            class="fst-italic text-center"
                        >
                            {$t("common.no-content")}
                        </td>
                    </tr>
                {/if}
            </tbody>
            <tfoot>
                <tr>
                    <td colspan="1">
                        <Button
                            type="button"
                            color="primary"
                            outline
                            size="sm"
                            onclick={() => val.s.push("")}
                        >
                            <Icon name="plus" />
                            {$t("common.new-row")}
                        </Button>
                    </td>
                </tr>
            </tfoot>
        </table>

        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th>Flags</th>
                </tr>
            </thead>
            <tbody>
                {#if val.t && val.t.length}
                    {#each val.t as rua, idx}
                        <tr>
                            <td>
                                <Input
                                    size="sm"
                                    bind:value={val.t[idx]}
                                />
                            </td>
                            <td>
                                <Button
                                    type="button"
                                    color="danger"
                                    outline
                                    size="sm"
                                    onclick={() => val.t.splice(idx, 1)}
                                >
                                    <Icon name="trash" />
                                </Button>
                            </td>
                        </tr>
                    {/each}
                {:else}
                    <tr>
                        <td
                            colspan={2}
                            class="fst-italic text-center"
                        >
                            {$t("common.no-content")}
                        </td>
                    </tr>
                {/if}
            </tbody>
            <tfoot>
                <tr>
                    <td colspan="1">
                        <Button
                            type="button"
                            color="primary"
                            outline
                            size="sm"
                            onclick={() => val.t.push("")}
                        >
                            <Icon name="plus" />
                            {$t("common.new-row")}
                        </Button>
                    </td>
                </tr>
            </tfoot>
        </table>
    </form>

</div>
