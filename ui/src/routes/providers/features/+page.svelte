<script lang="ts">
 import {
     Container,
     Icon,
     Table,
     Spinner,
 } from 'sveltestrap';

 import { listProviders } from '$lib/api/provider_specs';
 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import { t } from '$lib/translations';

 const capabilities = [
     "ListDomains",
     "rr-1-A",
     "rr-257-CAA",
     "rr-61-OPENPGPKEY",
     "rr-12-PTR",
     "rr-33-SRV",
     "rr-44-SSHFP",
     "rr-52-TLSA",
 ];
</script>

<Container class="d-flex flex-column flex-fill" fluid>
    {#await listProviders()}
        <Spinner size="lg" />
    {:then providers}
        <div
            style="overflow-x: scroll"
        >
        <Table
            hover
        >
            <thead>
                <tr>
                    <th>Fournisseurs</th>
                    {#each capabilities as cap}
                        <th class="text-center" style="white-space: nowrap;">
                            {#if cap == 'rr-1-A'}
                                {$t('record.common-records')}
                            {:else if cap.startsWith('rr-')}
                                {$t('common.records', { n: 2, type: cap.slice(cap.lastIndexOf('-')+1) })}
                            {:else}
                                {$t('provider.capability.' + cap, { default: cap })}
                            {/if}
                        </th>
                    {/each}
                </tr>
            </thead>
            <tbody>
                {#each Object.keys(providers) as ptype (ptype)}
                    {@const provider = providers[ptype]}
                    <tr>
                        <td class="text-center">
                            <ImgProvider
                                {ptype}
                                style="max-width: 100%; max-height: 2.5em"
                            /><br>
                            <strong>
                                {provider.name}
                            </strong>
                        </td>
                        {#each capabilities as cap}
                            <td
                                class="align-middle text-center"
                                class:table-danger={!provider.capabilities.includes(cap)}
                                class:table-success={provider.capabilities.includes(cap)}
                            >
                                {#if provider.capabilities.includes(cap)}
                                    <Icon name="check-lg" />
                                {:else}
                                    <Icon name="x-lg" />
                                {/if}
                            </td>
                        {/each}
                    </tr>
                {/each}
            </tbody>
        </Table>
        </div>
    {/await}
</Container>
