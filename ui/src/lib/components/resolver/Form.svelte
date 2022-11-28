<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Collapse,
     Container,
     FormGroup,
     Input,
     Spinner
 } from 'sveltestrap';

 import SelectType from '$lib/components/resolver/SelectType.svelte';
 import SelectResolver from '$lib/components/resolver/SelectResolver.svelte';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let value = {
     domain: "",
     type: "ANY",
     resolver: "local",
     custom: "",
 };
 export let showDNSSEC = false;

 export let sortedDomains = [];
 export let request_pending = false;

 function submitRequest() {
     request_pending = true;
     dispatch('submit', {value, showDNSSEC});
 }
</script>

<form class="pt-3 pb-5" on:submit|preventDefault={submitRequest}>
    <FormGroup>
        <label for="domain">
            {$t('common.domain')}
        </label>
        <Input
            aria-describedby="domainHelpBlock"
            id="domain"
            list="my-domains"
            required
            placeholder="happydomain.org"
            bind:value={value.domain}
        />
        <div id="domainHelpBlock" class="form-text">
            {$t('resolver.domain-description')}
        </div>
        <datalist id="my-domains">
            {#each sortedDomains as dn (dn.id)}
                <option>
                    {dn.domain}
                </option>
            {/each}
        </datalist>
    </FormGroup>

    <div class="text-center mb-3">
        <Button type="button" color="secondary" id="settingsToggler">
            {$t('resolver.advanced')}
        </Button>
    </div>

    <Collapse toggler="#settingsToggler">
        <FormGroup>
            <label for="select-type">
                {$t('common.field')}
            </label>
            <SelectType
                aria-describedby="typeHelpBlock"
                id="select-type"
                required
                bind:value={value.type}
            />
            <div id="typeHelpBlock" class="form-text">
                {$t('resolver.field-description')}
            </div>
        </FormGroup>

        <FormGroup>
            <label for="select-resolver">
                {$t('common.resolver')}
            </label>
            <SelectResolver
                aria-describedby="resolverHelpBlock"
                id="select-resolver"
                required
                bind:value={value.resolver}
            />
            <div id="resolverHelpBlock" class="form-text">
                {$t('resolver.resolver-description')}
            </div>
        </FormGroup>

        {#if value.resolver === "custom"}
            <FormGroup>
                <label for="custom-resolver">
                    {$t('resolver.custom')}
                </label>
                <Input
                    aria-describedby="customResolverHelpBlock"
                    id="custom-resolver"
                    required={value.resolver === 'custom'}
                    placeholder="127.0.0.1"
                    bind:value={value.custom}
                />
                <div id="customResolverHelpBlock" class="form-text">
                    {$t('resolver.custom-description')}
                </div>
            </FormGroup>
        {/if}

        <Input
            type="checkbox"
            label={$t('resolver.showDNSSEC')}
            id="showDNSSEC"
            bind:value={showDNSSEC}
            name="showDNSSEC"
            class="mb-3"
        />
    </Collapse>

    <div class="ml-3 mr-3">
        <Button
            type="submit"
            class="float-end"
            color="primary"
            disabled={request_pending}
        >
            {#if request_pending}
                <Spinner label={$t('common.spinning')} size="sm" />
            {/if}
            {$t('common.run')}
        </Button>
    </div>
</form>
