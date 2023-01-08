<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Input,
     InputGroup,
     InputGroupText,
     Modal,
     ModalBody,
 } from 'sveltestrap';

 import ModalFooter from '$lib/components/domains/ModalFooter.svelte';
 import ModalHeader from '$lib/components/domains/ModalHeader.svelte';
 import { validateDomain } from '$lib/dns';
 import type { Domain } from '$lib/model/domain';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let isOpen = false;
 const toggle = () => (isOpen = !isOpen);

 $: {
     if (isOpen) {
         value = "";
     }
 }

 export let origin: Domain;
 export let value: string = "";

 let newDomainState: boolean | undefined = undefined;
 $: newDomainState = value?validateNewSubdomain(value):undefined;

 let endsWithOrigin = false;
 $: endsWithOrigin = value.length > origin.domain.length && (
     value.substring(value.length - origin.domain.length) === origin.domain ||
     value.substring(value.length - origin.domain.length + 1) === origin.domain.substring(0, origin.domain.length - 1)
 )

 let newDomainAppend: string | null = null;
 $: {
     if (endsWithOrigin) {
         newDomainAppend = null;
     } else if (value.length > 0) {
         newDomainAppend = '.' + origin.domain;
     } else {
         newDomainAppend = origin.domain;
     }
 }

 function validateNewSubdomain(value: string): boolean | undefined {
     newDomainState = validateDomain(
         value,
         (value.length > origin.domain.length && value.substring(value.length - origin.domain.length) === origin.domain)?origin.domain:""
     );
     return newDomainState;
 }

 function submitSubdomainForm() {
     if (validateNewSubdomain(value)) {
         toggle();
         dispatch("show-next-modal", value);
     }
 }
</script>

<Modal
    {isOpen}
    {toggle}
>
    <ModalHeader {toggle} dn={origin.domain} />
    <ModalBody>
        <form
            id="addSubdomainForm"
            on:submit|preventDefault={submitSubdomainForm}
        >
            <p>
                {$t('domains.form-new-subdomain')}
                <span class="font-monospace">{origin.domain}</span>
                <InputGroup>
                    <Input
                        autofocus
                        class="font-monospace"
                        placeholder={$t('domains.placeholder-new-sub')}
                        invalid={newDomainState === false}
                        valid={newDomainState === true}
                        bind:value={value}
                    />
                    {#if newDomainAppend}
                        <InputGroupText class="font-monospace">
                            {newDomainAppend}
                        </InputGroupText>
                    {/if}
                </InputGroup>
            </p>
        </form>
    </ModalBody>
    <ModalFooter
        canContinue={newDomainState === true}
        form="addSubdomainForm"
        step={0}
        {toggle}
    />
</Modal>
