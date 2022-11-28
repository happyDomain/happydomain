<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Container,
     Col,
     Row,
 } from 'sveltestrap';

 import { resolve } from '$lib/api/resolver';
 import ResolverForm from '$lib/components/resolver/Form.svelte';
 import { nsttl, nsrrtype } from '$lib/dns';
 import { recordsFields } from '$lib/resolver';
 import { t } from '$lib/translations';
 import { toasts } from '$lib/stores/toasts';

 export let data = { };
 let question = null;
 let responses = null;
 let request_pending = false;

 $: {
     if (!data.form) {
         data.form = { };
     }
     data.form.domain = data.domain;

     resolve(data.form)
     .then(
         (response) => {
             question = Object.assign({ }, data.form)
             if (response.Answer) {
                 responses = response.Answer
             } else {
                 responses = 'no-answer'
             }
             request_pending = false
         },
         (error) => {
             toasts.addErrorToast({
                 title: $t('errors.resolve'),
                 message: error,
                 timeout: 5000,
             })
             request_pending = false
     })
 }

 function filteredResponses(responses, showDNSSEC) {
     if (!responses) {
         return [];
     }

     if (showDNSSEC) {
         return responses
     } else {
         return responses.filter(rr => (rr.Hdr.Rrtype !== 46 && rr.Hdr.Rrtype !== 47 && rr.Hdr.Rrtype !== 50))
     }
 }

 function responseByType(filteredResponses) {
     const ret = { };

     for (const i in filteredResponses) {
         if (!ret[filteredResponses[i].Hdr.Rrtype]) {
             ret[filteredResponses[i].Hdr.Rrtype] = []
         }
         ret[filteredResponses[i].Hdr.Rrtype].push(filteredResponses[i])
     }
     return ret;
 }

 function resolveDomain(event) {
     const form = event.detail.value;
     const showDNSSEC = event.detail.showDNSSEC;

     request_pending = true;
     goto('/resolver/' + encodeURIComponent(form.domain), {
         state: {form, showDNSSEC},
         noScroll: true,
     });
 }
</script>

<Container fluid class="flex-fill d-flex flex-column">
    <Row class="flex-grow-1">
        <Col md={{offset: 0, size: 4}} class="bg-light pt-3 pb-5">
        <div class="pt-2 sticky-top">
            <h1 class="text-center mb-3">
                {$t('menu.dns-resolver')}
            </h1>
            <ResolverForm
                bind:request_pending={request_pending}
                value={data.form}
                on:submit={resolveDomain}
            />
        </div>
        </Col>
        {#if responses === 'no-answer'}
            <Col md="8" class="pt-2">
                <h3>{$t('common.records', { number: 0, type: question.type })}</h3>
            </Col>
        {:else if responses}
            <Col md="8" class="pt-2">
                {@const resByType = responseByType(filteredResponses(responses, data.showDNSSEC))}
                {#each Object.keys(resByType) as type}
                    {@const rrs = resByType[type]}
                    <div>
                        <h3>{$t('common.records', { number: rrs.length, type: nsrrtype(type) })}</h3>
                        <table class="table table-hover table-sm">
                            <thead>
                                <tr>
                                    {#each recordsFields(Number(type)) as field}
                                        <th>
                                            {$t('record.' + field)}
                                        </th>
                                    {/each}
                                    <th>
                                        {$t('resolver.ttl')}
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each rrs as record}
                                    <tr>
                                        {#each recordsFields(Number(type)) as field}
                                            <td>
                                                {record[field]}
                                            </td>
                                        {/each}
                                        <td>
                                            {nsttl(Number(record.Hdr.Ttl))}
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/each}
            </Col>
        {/if}
    </Row>
</Container>
