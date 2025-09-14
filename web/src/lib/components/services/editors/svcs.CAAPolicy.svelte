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

    import { getRrtype, newRR, type dnsTypeCAA } from "$lib/dns_rr";
    import TableInput from "$lib/components/inputs/table.svelte";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import ResourceRawInput from "$lib/components/inputs/raw.svelte";
    import CAAIssuer from "./CAA-issuer.svelte";
    import CAAIodef from "./CAA-iodef.svelte";
    import { servicesSpecs } from "$lib/stores/services";
    import { t } from "$lib/translations";

    import issuers from "./CAA-issuers";

    const dispatch = createEventDispatcher();

    interface Props {
        readonly?: boolean;
        value: any;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();

    function newCAArr(tag: string, val: string): dnsTypeCAA {
        const rr = newRR(dn, getRrtype("CAA")) as dnsTypeCAA;

        rr.Tag = tag;
        rr.Value = val;

        return rr;
    }

    class CAAPolicy {
        records: Array<dnsTypeCAA>;
        DisallowIssue: boolean;
        DisallowWildcardIssue: boolean;
        DisallowMailIssue: boolean;

        constructor(records: Array<any>) {
            this.records = records["caa"];
            this.refreshDisallowIssue();
        }

        hasDisallowIssue(tag: string): boolean {
            for (const record of this.records) {
                if (record.Tag == tag && record.Value.trim() == ";") {
                    return true;
                }
            }

            return false;
        }

        refreshDisallowIssue(): boolean {
            this.DisallowIssue = this.hasDisallowIssue("issue");
            this.DisallowWildcardIssue = this.hasDisallowIssue("issuewild");
            this.DisallowMailIssue = this.hasDisallowIssue("issuemail");
        }

        changeDisallowIssue(tag: string) {
            return (e: Event) => {
                if (e.target.checked) {
                    this.records.push(newCAArr(tag, ";"));
                    this.refreshDisallowIssue();
                } else {
                    for (let i = this.records.length - 1; i >= 0; i--) {
                        const r = this.records[i];
                        if (r.Tag == tag && r.Value.trim() == ";") {
                            this.records.splice(i, 1);
                        }
                    }
                    this.refreshDisallowIssue();
                }
            };
        }
    }

    function addIssuer(tag: string) {
        return (e: Event) => {
            value["caa"].push(newCAArr(tag, e.detail));
        };
    }

    let val = $derived(new CAAPolicy(value));

    const type = "svcs.CAAPolicy";
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}

{#each value["caa"] as caa, i}
    <RecordLine {dn} {origin} bind:rr={value["caa"][i]} />
{/each}

<h4 class="mt-4">{$t("resources.CAA.title")}</h4>

<FormGroup>
    <Input
        id="issuedisabled"
        type="checkbox"
        label={$t("resources.CAA.no-issuers-hint")}
        checked={val.DisallowIssue}
        on:change={val.changeDisallowIssue("issue")}
    />
</FormGroup>

<h5>
    {$t("resources.CAA.auth-issuers")}
</h5>

{#if !val.DisallowIssue}
    <ul>
        {#if val.records.filter((r) => r.Tag == "issue").length}
            {#each val.records as issue, k}
                {#if issue.Tag == "issue"}
                    <li class="mb-3">
                        <CAAIssuer
                            {readonly}
                            bind:flag={val.records[k].Flag}
                            bind:tag={val.records[k].Tag}
                            bind:value={val.records[k].Value}
                            on:delete-issuer={() => { val.records.splice(k, 1); }}
                        />
                    </li>
                {/if}
            {/each}
        {:else}
            <Alert color="warning" fade={false}>
                <strong>{$t("resources.CAA.all-issuers-title")}</strong>
                {$t("resources.CAA.all-issuers-body")}
            </Alert>
        {/if}
        {#if !readonly}
            <li style:list-style="'+ '">
                <CAAIssuer
                    newone
                    on:add-issuer={addIssuer("issue")}
                />
            </li>
        {/if}
    </ul>
{:else}
    <Alert color="danger" fade={false}>
        <strong>{$t("resources.CAA.no-issuers-title")}</strong>
        {$t("resources.CAA.no-issuers-body")}
    </Alert>
{/if}

<h4 class="mt-4">{$t("resources.CAA.wild-issuers")}</h4>

<FormGroup>
    <Input
        id="wildcardissuedisabled"
        type="checkbox"
        label={$t("resources.CAA.no-wild-hint")}
        checked={val.DisallowWildcardIssue}
        on:change={val.changeDisallowIssue("issuewild")}
    />
</FormGroup>

<h5>
    {$t("resources.CAA.auth-issuers")}
</h5>

{#if !val.DisallowWildcardIssue}
    <ul>
        {#if val.records.filter((r) => r.Tag == "issuewild").length}
            {#each val.records as issue, k}
                {#if issue.Tag == "issuewild"}
                    <li class="mb-3">
                        <CAAIssuer
                            {readonly}
                            bind:flag={val.records[k].Flag}
                            bind:tag={val.records[k].Tag}
                            bind:value={val.records[k].Value}
                            on:delete-issuer={() => { val.records.splice(k, 1); }}
                        />
                    </li>
                {/if}
            {/each}
        {:else if val.DisallowIssue}
            <Alert color="danger" fade={false}>
                <strong>{$t("resources.CAA.no-issuers-title")}</strong>
                {$t("resources.CAA.no-wild-body")}
            </Alert>
        {:else if val.Issue}
            <Alert color="warning" fade={false}>
                <strong>{$t("resources.CAA.wild-same-title")}</strong>
                {$t("resources.CAA.wild-same-body")}
            </Alert>
        {:else}
            <Alert color="warning" fade={false}>
                <strong>{$t("resources.CAA.all-issuers-title")}</strong>
                {$t("resources.CAA.all-wild-issuers-body")}
            </Alert>
        {/if}
        {#if !readonly}
            <li style:list-style="'+ '">
                <CAAIssuer
                    newone
                    on:add-issuer={addIssuer("issuewild")}
                />
            </li>
        {/if}
    </ul>
{:else}
    <Alert color="danger" fade={false}>
        <strong>{$t("resources.CAA.no-wild-title")}</strong>
        {$t("resources.CAA.no-wild-body")}
    </Alert>
{/if}

<h4 class="mt-4">{$t("resources.CAA.mail-issuers")}</h4>

<FormGroup>
    <Input
        id="mailissuedisabled"
        type="checkbox"
        label={$t("resources.CAA.no-mail-hint")}
        checked={val.DisallowMailIssue}
        on:change={val.changeDisallowIssue("issuemail")}
    />
</FormGroup>

{#if !val.DisallowMailIssue && !val.records.filter((r) => r.Tag == "issuemail").length}
    <Alert color="warning" fade={false}>
        <strong>{$t("resources.CAA.mail-all-allowed-title")}</strong>
        {$t("resources.CAA.mail-all-allowed-body")}
    </Alert>
{/if}

<h5>
    {$t("resources.CAA.auth-issuers")}
</h5>

{#if !val.DisallowMailIssue}
    <ul>
        {#if val.records.filter((r) => r.Tag == "issuemail").length}
            {#each val.records as issue, k}
                {#if issue.Tag == "issuemail"}
                    <li class="mb-3">
                        <CAAIssuer
                            {readonly}
                            bind:flag={val.records[k].Flag}
                            bind:tag={val.records[k].Tag}
                            bind:value={val.records[k].Value}
                            on:delete-issuer={() => { val.records.splice(k, 1); }}
                        />
                    </li>
                {/if}
            {/each}
        {/if}
        {#if !readonly}
            <li style:list-style="'+ '">
                <CAAIssuer
                    newone
                    on:add-issuer={addIssuer("issuemail")}
                />
            </li>
        {/if}
    </ul>
{:else}
    <Alert color="danger" fade={false}>
        <strong>{$t("resources.CAA.no-mail-title")}</strong>
        {$t("resources.CAA.no-mail-body")}
    </Alert>
{/if}

<h4 class="mt-4">{$t("resources.CAA.incident-response")}</h4>

<p>
    {$t("resources.CAA.incident-response-text")}
</p>

{#if val.records.filter((r) => r.Tag == "iodef").length}
    {#each val.records as issue, k}
        {#if issue.Tag == "iodef"}
            <CAAIodef
                {readonly}
                bind:flag={val.records[k].Flag}
                bind:tag={val.records[k].Tag}
                bind:value={val.records[k].Value}
                on:delete-iodef={() => { val.records.splice(k, 1); }}
            />
        {/if}
    {/each}
{/if}
{#if !readonly}
    <CAAIodef
        newone
        on:add-iodef={addIssuer("iodef")}
    />
{/if}
