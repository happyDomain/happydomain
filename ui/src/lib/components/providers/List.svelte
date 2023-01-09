<script lang="ts">
 import { goto } from '$app/navigation';
 import { createEventDispatcher } from 'svelte';

 import {
     Badge,
     Dropdown,
     DropdownItem,
     DropdownMenu,
     DropdownToggle,
     Icon,
     Spinner,
 } from 'sveltestrap';

 import { deleteProvider } from '$lib/api/provider';
 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import HListGroup from '$lib/components/ListGroup.svelte';
 import type { Provider } from '$lib/model/provider';
 import { domains } from '$lib/stores/domains';
 import { providers, providersSpecs, refreshProviders, refreshProvidersSpecs } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let flush = false;
 export let noLabel = false;
 export let noDropdown = false;
 export let selectedProvider: Provider|null = null;
 export let items: Array<any>;

 if (!$providersSpecs) refreshProvidersSpecs();

 let domain_in_providers: Record<string, number> = {};
 $: {
     if ($domains && $providers) {
         const tmp: Record<string, number> = { };

         for (const p of $providers) {
             tmp[p._id] = 0;
         }

         for (const domain of $domains) {
             if (!tmp[domain.id_provider]) {
                 tmp[domain.id_provider] = 0;
             }
             tmp[domain.id_provider]++;
         }

         domain_in_providers = tmp;
     }
 }

 function selectProvider(event: CustomEvent<Provider>) {
     if (selectedProvider && selectedProvider._id == event.detail._id) {
         selectedProvider = null;
     } else {
         selectedProvider = event.detail;
         dispatch("click", selectedProvider);
     }
 }

 function updateProvider(event: Event, item: Provider) {
     event.stopPropagation();
     goto('/providers/' + encodeURIComponent(item._id))
 }

 async function delProvider(event: Event, item: Provider) {
     event.stopPropagation();
     await deleteProvider(item._id);
     refreshProviders();
 }
</script>

{#if !items || $providersSpecs == null}
    <div class="d-flex gap-2 align-items-center justify-content-center my-3">
        <Spinner color="primary" label="Spinning" class="mr-3" /> {$t("wait.retrieving-providers")}
    </div>
{:else}
    <HListGroup
        button
        {items}
        {flush}
        {...$$restProps}
        isActive={(item) => (selectedProvider != null && item._id == selectedProvider._id)}
        on:click={selectProvider}
        let:item={item}
    >
        <div slot="empty">
            <form on:submit|preventDefault={() => dispatch('new-provider')}>
                {@html $t('provider.empty', {"action": `<button type="submit" class="btn btn-link p-0">${$t('provider.empty-action')}</button>`})}
            </form>
        </div>
        <div class="d-flex flex-fill justify-content-between" style="max-width: 100%">
        <div class="d-flex" style="min-width: 0">
            <div class="text-center" style="width: 50px;">
                <ImgProvider ptype={item._srctype} style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em" />
            </div>
            {#if item._comment}
                <div class="text-truncate" title={item._comment}>
                    {item._comment}
                </div>
            {:else}
                <em>{$t('provider.no-name')}</em>
            {/if}
        </div>
        {#if !(noLabel && noDropdown)}
            <div class="d-flex">
                {#if !noLabel}
                    <div>
                        <Badge
                            class="mx-1"
                            color={domain_in_providers[item._id] > 0 ? 'success' : 'danger'}
                        >
                            {$t('provider.associations', {"count": domain_in_providers[item._id]})}
                        </Badge>
                        {#if $providersSpecs[item._srctype]}
                            <Badge
                                class="mx-1"
                                color="secondary"
                                title={item._srctype}
                            >
                                {$providersSpecs[item._srctype].name}
                            </Badge>
                        {/if}
                    </div>
                {/if}
                {#if !noDropdown}
                    <Dropdown
                        size="sm"
                        style="margin-right: -10px"
                    >
                        <DropdownToggle
                            color="link"
                            on:click={(event) => event.stopPropagation()}
                        >
                            <Icon name="three-dots" />
                        </DropdownToggle>
                        <DropdownMenu>
                            <DropdownItem on:click={(e) => updateProvider(e, item)}>
                                {$t('provider.update')}
                            </DropdownItem>
                            <DropdownItem on:click={(e) => delProvider(e, item)}>
                                {$t('provider.delete')}
                            </DropdownItem>
                        </DropdownMenu>
                    </Dropdown>
                {/if}
            </div>
        {/if}
        </div>
    </HListGroup>
{/if}
