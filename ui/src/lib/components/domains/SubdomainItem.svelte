<script lang="ts">
 import {
     Badge,
     Button,
     Icon,
     Popover,
 } from 'sveltestrap';

 import Service from '$lib/components/domains/Service.svelte';
 import { fqdn } from '$lib/dns';
 import type { Domain } from '$lib/model/domain';
 import type { ServiceCombined } from '$lib/model/service';
 import { ZoneViewGrid } from '$lib/model/usersettings';
 import { userSession } from '$lib/stores/usersession';
 import { t } from '$lib/translations';

 export let aliases: Array<string> = [];
 export let dn: string;
 export let origin: Domain;
 export let showSubdomainsList = false;
 export let services: Array<ServiceCombined>;
 export let zoneId: string;

 let showResources = true;

 function isCNAME() {
     return services.length === 1 && services[0]._svctype === 'svcs.CNAME'
 }
</script>

{#if isCNAME()}
    <div>
        <h2
            id={dn}
            class="sticky-top"
            style="background: white; z-index: 1"
        >
            <span style="white-space: nowrap">
                <Icon name="link" />
                <span
                    class="font-monospace"
                    title={fqdn(dn, origin.domain)}
                >
                    {fqdn(dn, origin.domain)}
                </span>
            </span>
            <span style="white-space: nowrap">
                <Icon name="arrow-right" />
                <span class="font-monospace">
                    {services[0].Service.Target}
                </span>
            </span>
            <Button
                color="primary"
                size="sm"
                class="ml-2"
                click="$emit('add-new-service', dn)"
            >
                <Icon name="plus" />
                {$t('service.add')}
            </Button>
            <Button
                color="info"
                outline
                size="sm"
                class="ml-2"
                click="$emit('show-service-window', zoneServices[0])"
            >
                <Icon name="pencil" />
                {$t('domains.edit-target')}
            </Button>
            <Button
                color="danger"
                outline
                size="sm"
                class="ml-2"
                click="deleteCNAME()"
            >
                <Icon name="x-circle" />
                {$t('domains.drop-alias')}
            </Button>
        </h2>
    </div>
{:else}
    <div>
        <div
            class="d-flex align-items-center sticky-top mb-2 gap-2"
            style="background: white; z-index: 1"
        >
            <h2
                id={dn?dn:'@'}
                style="white-space: nowrap; cursor: pointer;"
                class="mb-0"
                on:click={() => showResources = !showResources}
                on:keypress={() => showResources = !showResources}
            >
                {#if showResources}
                    <Icon name="chevron-down" />
                {:else}
                    <Icon name="chevron-right" />
                {/if}
                <span
                    class="font-monospace"
                    title={fqdn(dn, origin.domain)}
                >
                    {fqdn(dn, origin.domain)}
                </span>
            </h2>
            {#if aliases.length != 0}
                <Badge
                    id={"popoverbadge-" + dn.replace('.', '__')}
                    style="cursor: pointer;"
                >
                    + {$t('domains.n-aliases', {n: aliases.length})}
                </Badge>
                <Popover
                    dismissible
                    placement="bottom"
                    target={"popoverbadge-" + dn.replace('.', '__')}
                    class="font-monospace"
                >
                    {#each aliases as alias}
                        <a href={"#" + alias}>
                            {alias}
                        </a>
                        <br>
                    {/each}
                </Popover>
            {/if}
            {#if $userSession && $userSession.settings.zoneview !== ZoneViewGrid}
                <Button
                    color="primary"
                    size="sm"
                    click="$emit('add-new-service', dn)"
                >
                    <Icon name="plus" />
                    {$t('domains.add-a-service')}
                </Button>
            {/if}
            <Button
                color="primary"
                outline
                size="sm"
                click="$emit('add-new-alias', dn)"
            >
                <Icon name="link" />
                {$t('domains.add-an-alias')}
            </Button>
            {#if !showSubdomainsList && !dn}
                <Button
                    color="secondary"
                    outline
                    size="sm"
                    click="$emit('add-subdomain')"
                >
                    <Icon name="server" />
                    {$t('domains.add-a-subdomain')}
                </Button>
            {/if}
        </div>
        {#if showResources}
            <div
                class:d-flex={showResources && $userSession && $userSession.settings.zoneview === ZoneViewGrid}
                class:justify-content-around={showResources && $userSession && $userSession.settings.zoneview === ZoneViewGrid}
                class:flex-wrap={showResources && $userSession && $userSession.settings.zoneview === ZoneViewGrid}
            >
                {#each services as service}
                    <Service
                        {origin}
                        {service}
                        {zoneId}
                    />
                {/each}
                {#if $userSession && $userSession.settings.zoneview === ZoneViewGrid}
                    <Service
                        {origin}
                        {services}
                        {zoneId}
                    />
                {/if}
            </div>
        {/if}
    </div>
{/if}
