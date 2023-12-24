<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Container,
     Col,
     Row,
     Table,
 } from 'sveltestrap';

 import { resolve as APIResolve } from '$lib/api/resolver';
 import ResolverForm from '$lib/components/resolver/Form.svelte';
 import { nsttl, nsrrtype } from '$lib/dns';
 import { recordsFields } from '$lib/resolver';
 import type { ResolverForm as ResolverFormT } from '$lib/model/resolver';
 import { t } from '$lib/translations';
 import { toasts } from '$lib/stores/toasts';

 export let data: {form?: ResolverFormT; domain: string; showDNSSEC: boolean;};
 let question: ResolverFormT | null = null;
 let responses: Array<any> | 'no-answer' | null = null;
 let error_response: string | null = null;
 let request_pending = false;

 function resolve(form: ResolverFormT) {
     if (!form.domain) return;

     APIResolve(form)
     .then(
         (response) => {
             error_response = null;
             question = Object.assign({ }, data.form)
             if (response.Answer) {
                 responses = response.Answer;
             } else {
                 responses = 'no-answer';
             }
             request_pending = false;
         },
         (error) => {
             responses = null;
             error_response = error;
             toasts.addErrorToast({
                 title: $t('errors.resolve'),
                 message: error,
                 timeout: 5000,
             });
             request_pending = false;
     });
 }

 $: {
     if (!data.form) {
         data.form = {domain: "", type: "ANY", resolver: "local"};
     }
     data.form.domain = data.domain;

     resolve(data.form);
 }

 function filteredResponses(responses: Array<any>, showDNSSEC: boolean): Array<any> {
     if (!responses) {
         return [];
     }

     if (showDNSSEC) {
         return responses;
     } else {
         return responses.filter(rr => (rr.Hdr.Rrtype !== 46 && rr.Hdr.Rrtype !== 47 && rr.Hdr.Rrtype !== 50));
     }
 }

 function responseByType(filteredResponses: Array<any>): Record<string, Array<any>> {
     const ret: Record<string, Array<any>> = { };

     for (const i in filteredResponses) {
         if (!ret[filteredResponses[i].Hdr.Rrtype]) {
             ret[filteredResponses[i].Hdr.Rrtype] = []
         }
         ret[filteredResponses[i].Hdr.Rrtype].push(filteredResponses[i])
     }
     return ret;
 }

 function resolveDomain(event: CustomEvent<{value: ResolverFormT; showDNSSEC: boolean;}>): void {
     const form = event.detail.value;
     const showDNSSEC = event.detail.showDNSSEC;

     request_pending = true;
     goto('/resolver/' + encodeURIComponent(form.domain), {
         state: {form, showDNSSEC},
         noScroll: true,
     });
 }
</script>

{#if data.domain}
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
        {#if error_response !== null}
            <Col md="8" class="pt-3">
                <h3 class="text-center text-danger">{error_response}</h3>
            </Col>
        {:else if responses === 'no-answer'}
            <Col md="8" class="pt-2">
                <h3>{$t('common.records', { n: 0, type: question?question.type:"-" })}</h3>
            </Col>
        {:else if responses != null}
            <Col md="8" class="pt-2">
                {@const resByType = responseByType(filteredResponses(/* @ts-ignore */ responses, data.showDNSSEC))}
                {#each Object.keys(resByType) as type}
                    {@const rrs = resByType[type]}
                    <div>
                        <h3>{$t('common.records', { n: rrs.length, type: nsrrtype(type) })}</h3>
                        <Table
                            size="sm"
                            hover
                        >
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
                        </Table>
                    </div>
                {/each}
            </Col>
        {/if}
    </Row>
</Container>
{:else}
<Container fluid class="d-flex flex-column">
    <Row class="flex-grow-1">
        <Col md={{offset: 2, size: 8}} class="pt-4 pb-5">
            <h1 class="text-center mb-3">
                {$t('menu.dns-resolver')}
            </h1>
            <ResolverForm
                bind:request_pending={request_pending}
                on:submit={resolveDomain}
            />
        </Col>
    </Row>
</Container>
{/if}
