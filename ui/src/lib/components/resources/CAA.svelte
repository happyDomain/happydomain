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
 import { createEventDispatcher } from 'svelte';

 import {
     Alert,
     Badge,
     Button,
     FormGroup,
     Icon,
     Input,
 } from '@sveltestrap/sveltestrap';

 import TableInput from '$lib/components/resources/table.svelte';
 import ResourceRawInput from '$lib/components/resources/raw.svelte';
 import CAAIssuer from '$lib/components/resources/CAA-issuer.svelte';
 import CAAIodef from '$lib/components/resources/CAA-iodef.svelte';
 import { t } from '$lib/translations';

 import issuers from './CAA-issuers';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let index: string;
 export let readonly = false;
 export let specs: any;
 export let value: any;
</script>

<h4>{$t("resources.CAA.title")}</h4>

<FormGroup>
    <Input id="issuedisabled" type="checkbox" label={$t("resources.CAA.no-issuers-hint")} bind:checked={value.DisallowIssue} />
</FormGroup>

<h5>
    {$t("resources.CAA.auth-issuers")}
</h5>

{#if !value.DisallowIssue}
    <ul>
        {#if value.Issue}
            {#each value.Issue as issue, k}
                <li class="mb-3">
                    <CAAIssuer
                        {readonly}
                        bind:value={value.Issue[k]}
                        on:delete-issuer={() => {value.Issue.splice(k, 1); value = value;}}
                    />
                </li>
            {/each}
        {:else}
            <Alert color="warning" fade={false}>
                <strong>{$t("resources.CAA.all-issuers-title")}</strong> {$t("resources.CAA.all-issuers-body")}
            </Alert>
        {/if}
        {#if !readonly}
            <li style:list-style="'+ '">
                <CAAIssuer
                    newone
                    on:add-issuer={(e) => {if (!value.Issue) value.Issue = []; value.Issue.push(e.detail); value = value;}}
                />
            </li>
        {/if}
    </ul>
{:else}
    <Alert color="danger" fade={false}>
        <strong>{$t("resources.CAA.no-issuers-title")}</strong> {$t("resources.CAA.no-issuers-body")}
    </Alert>
{/if}

<h4 class="mt-4">{$t("resources.CAA.wild-issuers")}</h4>

<FormGroup>
    <Input id="wildcardissuedisabled" type="checkbox" label={$t("resources.CAA.no-wild-hint")} bind:checked={value.DisallowWildcardIssue} />
</FormGroup>

<h5>
    {$t("resources.CAA.auth-issuers")}
</h5>

{#if !value.DisallowWildcardIssue}
    <ul>
        {#if value.IssueWild}
            {#each value.IssueWild as issue, k}
                <li class="mb-3">
                    <CAAIssuer
                        {readonly}
                        bind:value={value.IssueWild[k]}
                        on:delete-issuer={() => {value.IssueWild.splice(k, 1); value = value;}}
                    />
                </li>
            {/each}
        {:else if value.DisallowIssue}
            <Alert color="danger" fade={false}>
                <strong>{$t("resources.CAA.no-issuers-title")}</strong> {$t("resources.CAA.no-wild-body")}
            </Alert>
        {:else if value.Issue}
            <Alert color="warning" fade={false}>
                <strong>{$t("resources.CAA.wild-same-title")}</strong> {$t("resources.CAA.wild-same-body")}
            </Alert>
        {:else}
            <Alert color="warning" fade={false}>
                <strong>{$t("resources.CAA.all-issuers-title")}</strong> {$t("resources.CAA.all-wild-issuers-body")}
            </Alert>
        {/if}
        {#if !readonly}
            <li style:list-style="'+ '">
                <CAAIssuer
                    newone
                    on:add-issuer={(e) => {if (!value.IssueWild) value.IssueWild = []; value.IssueWild.push(e.detail); value = value;}}
                />
            </li>
        {/if}
    </ul>
{:else}
    <Alert color="danger" fade={false}>
        <strong>{$t("resources.CAA.no-wild-title")}</strong> {$t("resources.CAA.no-wild-body")}
    </Alert>
{/if}

<h4 class="mt-4">{$t("resources.CAA.mail-issuers")}</h4>

<FormGroup>
    <Input id="mailissuedisabled" type="checkbox" label={$t("resources.CAA.no-mail-hint")} bind:checked={value.DisallowMailIssue} />
</FormGroup>

{#if !value.DisallowMailIssue && !value.IssueMail}
    <Alert color="warning" fade={false}>
        <strong>{$t("resources.CAA.mail-all-allowed-title")}</strong> {$t("resources.CAA.mail-all-allowed-body")}
    </Alert>
{/if}

<h5>
    {$t("resources.CAA.auth-issuers")}
</h5>

{#if !value.DisallowMailIssue}
    <ul>
        {#if value.IssueMail}
            {#each value.IssueMail as issue, k}
                <li class="mb-3">
                    <CAAIssuer
                        {readonly}
                        bind:value={value.IssueMail[k]}
                        on:delete-issuer={() => {value.IssueMail.splice(k, 1); value = value;}}
                    />
                </li>
            {/each}
        {/if}
        {#if !readonly}
            <li style:list-style="'+ '">
                <CAAIssuer
                    newone
                    on:add-issuer={(e) => {if (!value.IssueMail) value.IssueMail = []; value.IssueMail.push(e.detail); value = value;}}
                />
            </li>
        {/if}
    </ul>
{:else}
    <Alert color="danger" fade={false}>
        <strong>{$t("resources.CAA.no-mail-title")}</strong> {$t("resources.CAA.no-mail-body")}
    </Alert>
{/if}

<h4 class="mt-4">{$t("resources.CAA.incident-response")}</h4>

<p>
    {$t("resources.CAA.incident-response-text")}
</p>

{#if value.Iodef}
    {#each value.Iodef as iodef, k}
        <CAAIodef
            {readonly}
            bind:value={value.Iodef[k]}
        />
    {/each}
{/if}
{#if !readonly}
    <CAAIodef
        newone
    />
{/if}
