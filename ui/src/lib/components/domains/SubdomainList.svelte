<script lang="ts">
 import {
     Button,
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
 } from 'sveltestrap';

 import SubdomainItem from '$lib/components/domains/SubdomainItem.svelte';
 import type { Domain } from '$lib/model/domain';
 import type { Zone } from '$lib/model/zone';
 import { t } from '$lib/translations';

 export let origin: Domain;
 export let showSubdomainsList: boolean;
 export let sortedDomains: Array<string>;
 export let zone: Zone;

 let aliases: Record<string, Array<string>>;
 $: {
     const tmp: Record<string, Array<string>> = { };

     for (const dn of sortedDomains) {
         zone.services[dn].forEach(function (svc) {
             if (svc._svctype === 'svcs.CNAME') {
                 if (!tmp[svc.Service.Target]) {
                     tmp[svc.Service.Target] = []
                 }
                 tmp[svc.Service.Target].push(dn)
             }
         })
     }

     aliases = tmp;
 }

 let modal = { };
</script>

{#each sortedDomains as dn}
    <SubdomainItem
        aliases={aliases[dn]?aliases[dn]:[]}
        {dn}
        {origin}
        {showSubdomainsList}
        zoneId={zone.id}
        services={zone.services[dn]?zone.services[dn]:[]}
    />
{/each}

{#if zone}
    <h-modal-service
        ref="modalService"
        :domain="domain"
        :my-services="myServices"
        :services="services"
        :zone-id="zoneId"
        update-my-services="updateMyServices"
    />
{/if}

<Modal
    id="modal-addAlias"
>
    <ModalHeader>
        {$t('domains.add-an-alias')}
    </ModalHeader>
    <ModalBody>
        {#if modal && modal.dn != null}
            <form id="addAliasForm" on:submit={handleModalAliasSubmit}>
                <i18n path="domains.alias-creation">
                    <span class="text-monospace">{fqdn(modal.dn, origin.domain)}</span>
                </i18n>
                <b-input-group :append="'.' + origin.domain">
                    <b-input v-model="modal.alias" autofocus class="text-monospace" placeholder="new.subdomain" :state="modal.newDomainState" update="validateNewAlias" />
                </b-input-group>
                <div v-show="modal.alias" class="mt-3 text-center">
                    {$t('domains.alias-creation-sample')}<br>
                    <span class="text-monospace text-no-wrap">{fqdn(modal.alias, origin.domain)}</span>
                    <b-icon class="mr-1 ml-1" icon="arrow-right" />
                    <span class="text-monospace text-no-wrap">{fqdn(modal.dn, origin.domain)}</span>
                </div>
            </form>
        {/if}
    </ModalBody>
    <ModalFooter>
        <Button
            color="secondary"
            outline
            click="cancel()"
        >
            {$t('common.cancel')}
        </Button>
        <Button
            color="primary"
            form="addAliasForm"
            type="submit"
        >
            {$t('domains.add-alias')}
        </Button>
    </ModalFooter>
</Modal>
