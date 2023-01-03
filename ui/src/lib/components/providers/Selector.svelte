<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     ListGroup,
     ListGroupItem,
     Spinner,
 } from 'sveltestrap';

 import { listProviders } from '$lib/api/provider_specs';
 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import type { ProviderInfos, ProviderList } from '$lib/model/provider';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let value: string | null = null;
 let isLoading = true;
 let providers: ProviderList = { };

 listProviders().then(
     (res) => {
         providers = res;
         isLoading = false;
     }
 );

 function selectProvider(provider: ProviderInfos, ptype: string) {
     value = ptype;
     dispatch("provider-selected", {provider, ptype});
 }
</script>

<ListGroup {...$$restProps}>
    {#if isLoading}
        <ListGroupItem class="d-flex justify-content-center align-items-center gap-2">
            <Spinner color="primary" label="Spinning" class="mr-3" /> {$t('wait.retrieving-provider')}
        </ListGroupItem>
    {/if}
    {#each Object.keys(providers) as ptype (ptype)}
        {@const provider = providers[ptype]}
        <ListGroupItem
            active={value === ptype}
            tag="button"
            class="d-flex"
            on:click={() => selectProvider(provider, ptype)}
        >
            <div
                class="align-self-center text-center"
                style="min-width:50px;width:50px;"
            >
                <ImgProvider
                    {ptype}
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
