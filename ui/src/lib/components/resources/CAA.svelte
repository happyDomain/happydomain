<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Alert,
     Badge,
     Button,
     FormGroup,
     Icon,
     Input,
 } from 'sveltestrap';

 import TableInput from '$lib/components/resources/table.svelte';
 import ResourceRawInput from '$lib/components/resources/raw.svelte';
 import CAAIssuer from '$lib/components/resources/CAA-issuer.svelte';
 import CAAIodef from '$lib/components/resources/CAA-iodef.svelte';

 import issuers from './CAA-issuers';

 const dispatch = createEventDispatcher();

 export let edit = false;
 export let index: string;
 export let readonly = false;
 export let specs: any;
 export let value: any;
</script>

<h4>Certificates issuance</h4>

<FormGroup>
    <Input id="issuedisabled" type="checkbox" label="Disallow any certificate issuance" bind:checked={value.DisallowIssue} />
</FormGroup>

<h5>
    Authorized Issuers
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
                <strong>All issuer authorized.</strong> With those parameters, all issuer can create certificate for this domain and subdomain.
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
        <strong>No issuer authorized.</strong> With those parameters, no issuer is allowed to create certificate for this subdomain.
    </Alert>
{/if}

<h4>Wildcard certificates issuance</h4>

<FormGroup>
    <Input id="wildcardissuedisabled" type="checkbox" label="Disallow wildcard certificate issuance" bind:checked={value.DisallowWildcardIssue} />
</FormGroup>

<h5>
    Authorized Issuers
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
                <strong>No issuer authorized.</strong> With those parameters, no issuer is authorized to create wildcard certificate for this domain and subdomain. But this can be override with the following settings:
            </Alert>
        {:else if value.Issue}
            <Alert color="warning" fade={false}>
                <strong>Same as regular certificate issuance.</strong> With those parameters, all issuer authorized for certificate issuance can also create wildcard certificate for this domain and subdomain.
            </Alert>
        {:else}
            <Alert color="warning" fade={false}>
                <strong>All issuer authorized.</strong> With those parameters, all issuer can create wildcard certificate for this domain and subdomain.
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
        <strong>No wildcard issuer authorized.</strong> With those parameters, no issuer is allowed to create wildcard certificate for this subdomain.
    </Alert>
{/if}

<h4>Incident Response</h4>

<p>
    How would you want to be contacted in case of violation of the current security policy?
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
