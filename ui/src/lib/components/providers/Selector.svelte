<script lang="ts">
 import {
     ListGroup,
     ListGroupItem,
     Spinner,
 } from 'sveltestrap';

 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import { listProviders } from '$lib/api/provider_specs';

 export let value = null;
 let isLoading = false;
 let providers = [];

 listProviders().then((res) => providers = res)
</script>

<ListGroup>
    {#if isLoading}
        <ListGroupItem class="d-flex justify-content-center align-items-center">
            <Spinner variant="primary" label="Spinning" class="mr-3" /> Retrieving usable providers...
        </ListGroupItem>
    {/if}
    {#each Object.keys(providers) as ptype (ptype)}
        {@const provider = providers[ptype]}
        <ListGroupItem
            active={value === provider.id}
            button
            class="d-flex"
            on:click={() => value = provider.id}
        >
            <div
                class="align-self-center text-center"
                style="min-width:50px;width:50px;"
            >
                <ImgProvider
                    {provider}
                    style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em"
                />
            </div>
            <div
                class="align-self-center"
                style="line-height: 1.1"
            >
                <strong>{provider.name}</strong> &ndash;
                <small class="text-muted" title={provider.description}>{provider.description}</small>
            </div>
        </ListGroupItem>
    {/each}
</ListGroup>
