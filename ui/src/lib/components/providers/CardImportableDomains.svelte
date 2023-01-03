<script lang="ts">
 import {
     Badge,
     Button,
     Card,
     CardHeader,
     Icon,
     ListGroupItem,
     Spinner,
 } from 'sveltestrap';

 import { addDomain } from '$lib/api/domains';
 import { listImportableDomains } from '$lib/api/provider';
 import ZoneList from '$lib/components/ZoneList.svelte';
 import type { Domain } from '$lib/model/domain';
 import type { Provider } from '$lib/model/provider';
 import { providersSpecs } from '$lib/stores/providers';
 import { domains_idx, refreshDomains } from '$lib/stores/domains';
 import { toasts } from '$lib/stores/toasts';
 import { t } from '$lib/translations';

 export let provider: Provider;

 let importableDomainsList: Array<string>|null = null;
 let discoveryError: string|null = null;
 export let noDomainsList = false;

 $: {
     importableDomainsList = null;
     discoveryError = null;
     noDomainsList = false;
     listImportableDomains(provider).then(
         (l) => importableDomainsList = l,
         (err) => {
             importableDomainsList = [];
             if (err.name == "ProviderNoDomainListingSupport") {
                 noDomainsList = true;
             } else {
                 discoveryError = err.message;
                 throw err;
             }
         }
     );
 }

 function haveDomain(name: string) {
     let domain = undefined;
     if (name[name.length-1] == ".") {
         domain = $domains_idx[name];
     } else {
         domain = $domains_idx[name+"."];
     }
     return domain !== undefined && domain.id_provider == provider._id;
 }

 async function importDomain(name: string) {
     addDomain(name, provider)
     .then(
         (domain) => {
             toasts.addToast({
                 title: $t('domains.attached-new'),
                 message: $t('domains.added-success', { domain: domain.domain }),
                 href: '/domains/' + domain.domain,
                 color: 'success',
                 timeout: 5000,
             });

             refreshDomains();
         },
         (error) => {
             throw error;
         }
     );
 }

 async function importAllDomains() {
     if (importableDomainsList) {
         for (const d of importableDomainsList) {
             if (!haveDomain(d)) {
                 await importDomain(d);
             }
         }
     }
 }

 function doDomainAction(dn: Domain) {

 }
</script>

<Card {...$$restProps}>
    {#if !noDomainsList && !discoveryError}
        <CardHeader>
            <div class="d-flex justify-content-between">
                <div>
                    {$t("provider.provider")}
                    <em>
                        {#if provider._comment}
                            {provider._comment}
                        {:else if $providersSpecs}
                            {$providersSpecs[provider._srctype].name}
                        {/if}
                    </em>
                </div>
                {#if importableDomainsList != null}
                    <Button
                        type="button"
                        color="secondary"
                        size="sm"
                        on:click={importAllDomains}
                    >
                        {$t('provider.import-domains')}
                    </Button>
                {/if}
            </div>
        </CardHeader>
    {/if}
    {#if importableDomainsList == null}
        <div class="d-flex justify-content-center align-items-center gap-2 my-3">
            <Spinner color="primary" /> {$t('wait.asking-domains')}
        </div>
    {:else}
        <ZoneList
            flush
            domains={importableDomainsList.map((dn) => ({domain: dn, id_provider: provider._id}))}
        >
            <div slot="badges" let:item={domain}>
                {#if domain.state}
                    <Badge class="ml-1" color={domain.state}>
                        {#if domain.state === 'success'}
                            <Icon name="check" />
                        {:else if domain.state === 'info'}
                            <Icon name="exclamation-circle" />
                        {:else if domain.state === 'warning'}
                            <Icon name="exclamation-triangle" />
                        {:else if domain.state === 'danger'}
                            <Icon name="exclamation-octagon" />
                        {/if}
                        {domain.message}
                    </Badge>
                {:else if haveDomain(domain.domain)}
                    <Badge class="ml-1" color="success">
                        <Icon name="check" />
                        {$t('service.already')}
                    </Badge>
                {:else}
                    <Button
                        type="button"
                        class="ml-1"
                        color="primary"
                        size="sm"
                        disabled={domain.wait}
                        on:click={() => importDomain(domain.domain)}
                    >
                        {$t('domains.add-now')}
                    </Button>
                {/if}
                {#if domain.btn}
                    <Button
                        type="button"
                        class="ms-1"
                        color={domain.state}
                        size="sm"
                        on:click={() => doDomainAction(domain)}
                    >
                        {$t(domain.btn)}
                    </Button>
                {/if}
            </div>
            <svelte:fragment slot="no-domain">
                {#if discoveryError}
                    <ListGroupItem class="mx-2 my-3">
                        <p class="text-danger">
                            <Icon name="exclamation-octagon-fill" class="float-start display-5 me-2" />
                            {discoveryError}
                        </p>
                        <div class="text-center">
                            <Button
                                href={"/providers/" + encodeURIComponent(provider._id)}
                                outline
                            >
                                {$t('provider.check-config')}
                            </Button>
                        </div>
                    </ListGroupItem>
                {:else if noDomainsList}
                    <ListGroupItem class="text-center my-3">
                        {$t('errors.domain-list')}
                    </ListGroupItem>
                {:else if !importableDomainsList || importableDomainsList.length === 0}
                    <ListGroupItem class="text-center my-3">
                        {$t('errors.domain-have')}
                    </ListGroupItem>
                {:else if importableDomainsList.length === 0}
                    <ListGroupItem class="text-center my-3">
                        {#if $providersSpecs}
                            <i18n path="errors.domain-all-imported">
                                {$providersSpecs[provider._srctype].name}
                            </i18n>
                        {/if}
                    </ListGroupItem>
                {/if}
            </svelte:fragment>
        </ZoneList>
    {/if}
</Card>
