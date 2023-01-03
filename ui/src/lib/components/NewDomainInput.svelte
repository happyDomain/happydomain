<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     Icon,
     Input,
     InputGroup,
     ListGroup,
     ListGroupItem,
     Spinner,
 } from 'sveltestrap';

 import { addDomain } from '$lib/api/domains';
 import { validateDomain } from '$lib/dns';
 import type { Provider } from '$lib/model/provider';
 import { refreshDomains } from '$lib/stores/domains';
 import { toasts } from '$lib/stores/toasts';
 import { t } from '$lib/translations';

 export let autofocus = false;
 export let provider: Provider|null = null;
 export let value = "";

 let addingNewDomain = false;
 let newDomainState: boolean|undefined = undefined;

 function addDomainToProvider() {
     addingNewDomain = true;

     if (!provider) {
         goto('/domains/' + encodeURIComponent(value) + '/new');
     } else {
         addDomain(value, provider)
         .then(
             (domain) => {
                 addingNewDomain = false;
                 toasts.addToast({
                     title: $t('domains.attached-new'),
                     message: $t('domains.added-success', { domain: domain.domain }),
                     href: '/domains/' + domain.domain,
                     color: 'success',
                     timeout: 5000,
                 });

                 value = "";
                 refreshDomains();
             },
             (error) => {
                 addingNewDomain = false;
                 throw error;
             }
         );
     }
 }

 function validateNewDomain(val: string|undefined): boolean|undefined {
     if (val) {
         newDomainState = validateDomain(val);
     } else {
         newDomainState = validateDomain(value);
     }

     return newDomainState;
 }

 function inputChange(event: Event) {
     if (event instanceof InputEvent) {
         validateNewDomain(event.data?value+event.data:value.substring(0,value.length-1));
     }
 }
</script>

<ListGroup {...$$restProps}>
    <form on:submit|preventDefault={addDomainToProvider}>
        <ListGroupItem class="d-flex justify-content-between align-items-center">
            <InputGroup>
                <label for="newdomaininput" class="text-center" style="width: 50px; font-size: 2.3rem">
                    <Icon name="plus" />
                </label>
                <Input
                    id="newdomaininput"
                    {autofocus}
                    class="font-monospace"
                    placeholder={$t('domains.placeholder-new')}
                    invalid={value.length ? newDomainState !== undefined && !newDomainState : undefined}
                    valid={value.length ? newDomainState : undefined}
                    style="border:none;box-shadow:none;z-index:0"
                    bind:value={value}
                    on:input={inputChange}
                />
                {#if value.length}
                    <Button
                        type="submit"
                        outline
                        color="primary"
                        disabled={addingNewDomain}
                    >
                        {#if addingNewDomain}
                            <Spinner size="sm" class="me-1" />
                        {/if}
                        {$t('common.add-new-thing', { thing: $t('domains.kind') })}
                    </Button>
                {/if}
            </InputGroup>
        </ListGroupItem>
    </form>
</ListGroup>
