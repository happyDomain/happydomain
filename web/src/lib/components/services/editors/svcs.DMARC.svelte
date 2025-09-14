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

<script lang="ts">
    import { createEventDispatcher } from "svelte";

    import { Alert, Badge, Button, FormGroup, Icon, Input } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { servicesSpecs } from "$lib/stores/services";
    import { parseKeyValueTxt } from "$lib/dns";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        dn: string;
        origin: Domain;
        value: any;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    function parseDMARC(val) {
        const dmarc = parseKeyValueTxt(val);

        dmarc.rua = dmarc.rua && dmarc.rua.length ? dmarc.rua.split(",") : [];
        dmarc.ruf = dmarc.ruf && dmarc.ruf.length ? dmarc.ruf.split(",") : [];
        dmarc.fo = dmarc.fo && dmarc.fo.length ? dmarc.fo.split(",") : [];
        dmarc.rf = dmarc.rf && dmarc.rf.length ? dmarc.rf.split(",") : [];

        return dmarc;
    }
    function stringifyDMARC(val) {
        const sep = (value["txt"].Txt.indexOf("; ") >= 0 ? "; " : ";");

        return "v=" + (val["v"] ? val["v"] : "DMARCv1") +
                  (val["p"] ? sep + "p=" + val["p"] : "") +
                  (val["sp"] ? sep + "sp=" + val["sp"] : "") +
                  (val["adkim"] ? sep + "adkim=" + val["adkim"] : "") +
                  (val["aspf"] ? sep + "aspf=" + val["aspf"] : "") +
                  (val["fo"] && val["fo"].length ? sep + "fo=" + val["fo"].join(",") : "") +
                  (val["rf"] && val["rf"].length ? sep + "rf=" + val["rf"].join(",") : "") +
                  (val["ri"] ? sep + "ri=" + val["ri"] : "") +
                  (val["rua"] && val["rua"].length ? sep + "rua=" + val["rua"].join(",") : "") +
                  (val["ruf"] && val["ruf"].length ? sep + "ruf=" + val["ruf"].join(",") : "") +
                  (val["pct"] ? sep + "pct=" + val["pct"] : "");
    }
    let val = $state(parseDMARC(value["txt"].Txt));

    $effect(() => {
        val = parseDMARC(value["txt"].Txt);
    });
    $effect(() => {
        value["txt"].Txt = stringifyDMARC(val);
    });

    const type = "svcs.DMARC";
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Domain-based Message Authentication, Reporting, and Conformance</h4>
    <RecordLine class="mb-4" {dn} {origin} bind:rr={value["txt"]} />

    <form id="addSvcForm">
        <BasicInput
            edit
            specs={{
                  id: "v",
                  label: "Version",
                  placeholder: "DMARCv1",
                  type: "string",
                  description: "Defines the version of DMARC to use",
                  }}
            bind:value={val.v}
        />

        <BasicInput
            edit
            specs={{
                  id: "p",
                  label: "Requested Mail Receiver policy",
                  type: "string",
                  description: "Indicates the policy to be enacted by the Receiver",
                  choices: ["none", "quarantine", "reject"],
                  default: "none",
                  }}
            bind:value={val.p}
        />

        <BasicInput
            edit
            specs={{
                  id: "sp",
                  label: "Requested Mail Receiver policy for all subdomains",
                  type: "string",
                  description: "Indicates the policy to be enacted by the Receiver when it receives mail for a subdomain",
                  choices: ["none", "quarantine", "reject"],
                  default: "none",
                  }}
            bind:value={val.sp}
        />

        <h4 class="mt-1 text-primary pb-1 border-bottom border-1">
            RUA <small class="text-muted">Addresses for aggregate feedback</small>
        </h4>
        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th>RUA</th>
                </tr>
            </thead>
            <tbody>
                {#if val.rua && val.rua.length}
                    {#each val.rua as rua, idx}
                        <tr>
                            <td>
                                <Input
                                    size="sm"
                                    bind:value={val.rua[idx]}
                                />
                            </td>
                            <td>
                                <Button
                                    type="button"
                                    color="danger"
                                    outline
                                    size="sm"
                                    onclick={() => val.rua.splice(idx, 1)}
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
                            onclick={() => val.rua.push("")}
                        >
                            <Icon name="plus" />
                            {$t("common.new-row")}
                        </Button>
                    </td>
                </tr>
            </tfoot>
        </table>

        <h4 class="mt-1 text-primary pb-1 border-bottom border-1">
            RUF <small class="text-muted">Addresses for message-specific failure information</small>
        </h4>

        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th>RUF</th>
                </tr>
            </thead>
            <tbody>
                {#if val.ruf && val.ruf.length}
                    {#each val.ruf as ruf, idx}
                        <tr>
                            <td>
                                <Input
                                    size="sm"
                                    bind:value={val.ruf[idx]}
                                />
                            </td>
                            <td>
                                <Button
                                    type="button"
                                    color="danger"
                                    outline
                                    size="sm"
                                    onclick={() => val.ruf.splice(idx, 1)}
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
                            onclick={() => val.ruf.push("")}
                        >
                            <Icon name="plus" />
                            {$t("common.new-row")}
                        </Button>
                    </td>
                </tr>
            </tfoot>
        </table>

        <FormGroup>
            <Input
                id="adkim"
                type="checkbox"
                label="Strict DKIM Alignment"
                checked={val.adkim == "s"}
                on:change={() => val.adkim = val.adkim == "s" ? "r" : "s"}
            />
        </FormGroup>

        <FormGroup>
            <Input
                id="aspf"
                type="checkbox"
                label="Strict SPF Alignment"
                checked={val.aspf == "s"}
                on:change={() => val.aspf = val.aspf == "s" ? "r" : "s"}
            />
        </FormGroup>

        <BasicInput
            edit
            specs={{
                  id: "ri",
                  label: "Interval between aggregate reports",
                  type: "time.Duration",
                  }}
            bind:value={val.ri}
        />

        <h4 class="mt-1 text-primary pb-1 border-bottom border-1">Failure reporting options
        </h4>
        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th>Failure reporting options</th>
                </tr>
            </thead>
            <tbody>
                {#if val.fo && val.fo.length}
                    {#each val.fo as fo, idx}
                        <tr>
                            <td>
                                <Input
                                    size="sm"
                                    bind:value={val.fo[idx]}
                                />
                            </td>
                            <td>
                                <Button
                                    type="button"
                                    color="danger"
                                    outline
                                    size="sm"
                                    onclick={() => val.fo.splice(idx, 1)}
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
                            onclick={() => val.fo.push("")}
                        >
                            <Icon name="plus" />
                            {$t("common.new-row")}
                        </Button>
                    </td>
                </tr>
            </tfoot>
        </table>

        <h4 class="mt-1 text-primary pb-1 border-bottom border-1">Format of the failure reports</h4>
        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th>Format of the failure reports</th>
                </tr>
            </thead>
            <tbody>
                {#if val.rf && val.rf.length}
                    {#each val.rf as rf, idx}
                        <tr>
                            <td>
                                <Input
                                    size="sm"
                                    bind:value={val.rf[idx]}
                                />
                            </td>
                            <td>
                                <Button
                                    type="button"
                                    color="danger"
                                    outline
                                    size="sm"
                                    onclick={() => val.rf.splice(idx, 1)}
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
                            onclick={() => val.rf.push("")}
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
                  id: "pct",
                  label: "Policy applies on",
                  placeholder: "100",
                  type: "number",
                  description: "Percentage of messages to which the DMARC policy is to be applied.",
                  }}
            bind:value={val.pct}
        />
    </form>
</div>
