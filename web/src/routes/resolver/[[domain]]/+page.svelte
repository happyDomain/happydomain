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
    import { navigate } from "$lib/stores/config";
    import { page } from "$app/state";
    import { untrack } from "svelte";

    import { Col, Container, Row, Table } from "@sveltestrap/sveltestrap";

    import { resolve as APIResolve } from "$lib/api/resolver";
    import { nsrrtype, nsttl } from "$lib/dns";
    import type { ResolverForm as ResolverFormT } from "$lib/model/resolver";
    import { recordsFields } from "$lib/resolver";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    import ResolverForm from "./Form.svelte";

    interface Props {
        data: { domain: string };
    }

    interface ResolverPageState {
        form?: ResolverFormT;
        showDNSSEC?: boolean;
    }

    let { data }: Props = $props();

    let domain = $derived(data.domain);
    let form: ResolverFormT = $state({ domain: "", type: "ANY", resolver: "local" });

    let error_response: string | null = $state(null);
    let request_pending = $state(false);
    let question: ResolverFormT | null = $state(null);
    let responses: Array<any> | "no-answer" | null = $state(null);

    function resolve(form: ResolverFormT) {
        if (!form.domain) return;

        APIResolve(form).then(
            (response) => {
                error_response = null;
                question = Object.assign({}, form);
                if (response.Answer) {
                    responses = response.Answer;
                } else {
                    responses = "no-answer";
                }
                request_pending = false;
            },
            (error) => {
                responses = null;
                error_response = error;
                toasts.addErrorToast({
                    title: $t("errors.resolve"),
                    message: error,
                    timeout: 5000,
                });
                request_pending = false;
            },
        );
    }

    $effect(() => {
        // Only track domain changes, not the entire data object
        if (domain) {
            untrack(() => {
                const state = page.state as ResolverPageState;
                if (state.form) {
                    form = Object.assign({}, state.form);
                } else {
                    form = { domain: "", type: "ANY", resolver: "local" };
                }

                form.domain = domain;

                resolve(form);
            });
        }
    });

    function filteredResponses(responses: Array<any>, showDNSSEC: boolean): Array<any> {
        if (!responses) {
            return [];
        }

        if (showDNSSEC) {
            return responses;
        } else {
            return responses.filter(
                (rr) => rr.Hdr.Rrtype !== 46 && rr.Hdr.Rrtype !== 47 && rr.Hdr.Rrtype !== 50,
            );
        }
    }

    function responseByType(filteredResponses: Array<any>): Record<string, Array<any>> {
        const ret: Record<string, Array<any>> = {};

        for (const i in filteredResponses) {
            if (!ret[filteredResponses[i].Hdr.Rrtype]) {
                ret[filteredResponses[i].Hdr.Rrtype] = [];
            }
            ret[filteredResponses[i].Hdr.Rrtype].push(filteredResponses[i]);
        }
        return ret;
    }

    function resolveDomain(
        event: CustomEvent<{ value: ResolverFormT; showDNSSEC: boolean }>,
    ): void {
        const form = event.detail.value;
        const showDNSSEC = event.detail.showDNSSEC;

        request_pending = true;

        if (form.domain === domain) {
            resolve(form);
        } else {
            navigate("/resolver/" + encodeURIComponent(form.domain), {
                state: { form, showDNSSEC },
                noScroll: true,
            });
        }
    }
</script>

<svelte:head>
    <title>
        {$t("menu.dns-resolver")}
        {domain ? domain : ""}
        - happyDomain</title
    >
</svelte:head>

{#if domain}
    <Container fluid class="flex-fill d-flex flex-column">
        <Row class="flex-grow-1">
            <Col md={{ offset: 0, size: 4 }} class="bg-light pt-3 pb-5">
                <div class="sticky-top">
                    <div class="mb-4">
                        <h1 class="display-6 fw-bold">
                            {$t("menu.dns-resolver")}
                        </h1>
                    </div>
                    <ResolverForm bind:request_pending value={form} on:submit={resolveDomain} />
                </div>
            </Col>
            {#if error_response !== null}
                <Col md="8" class="pt-3 pb-5">
                    <h2 class="display-7 fw-bold mt-3">
                        <i class="bi bi-exclamation-triangle"></i>
                        {$t("errors.resolve")}
                    </h2>
                    <p class="lead">
                        {$t("resolver.error-description")}
                    </p>
                    <div class="card border-danger">
                        <div class="card-body">
                            <div class="d-flex align-items-center">
                                <i class="bi bi-x-circle text-danger fs-3 me-3"></i>
                                <div>
                                    <h5 class="card-title text-danger mb-1">
                                        {$t("resolver.query-failed")}
                                    </h5>
                                    <p class="card-text mb-0">{error_response}</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </Col>
            {:else if responses === "no-answer"}
                <Col md="8" class="pt-3 pb-5">
                    <h2 class="display-7 fw-bold mt-3">
                        {$t("common.records", { n: 0, type: question ? question.type : "-" })}
                    </h2>
                    <p class="lead">
                        {$t("resolver.no-records-description")}
                    </p>
                    <div class="card">
                        <div class="card-body text-center py-5">
                            <i class="bi bi-inbox fs-1 text-muted"></i>
                            <p class="mt-3 mb-0 text-muted">
                                {$t("resolver.no-answer")}
                            </p>
                        </div>
                    </div>
                </Col>
            {:else if responses != null}
                <Col md="8" class="pt-3 pb-5">
                    {@const resByType = responseByType(
                        filteredResponses(
                            /* @ts-ignore */ responses,
                            (page.state as ResolverPageState).showDNSSEC ?? false,
                        ),
                    )}
                    <h2 class="display-7 fw-bold">
                        {$t("resolver.query-results")}
                    </h2>
                    <p class="lead mb-4">
                        {#if question}
                            {$t("resolver.results-description", {
                                domain: question.domain,
                                type: question.type,
                            })}
                        {:else}
                            {$t("resolver.results-description-default")}
                        {/if}
                    </p>
                    {#each Object.keys(resByType) as type}
                        {@const rrs = resByType[type]}
                        <div class="card mb-4">
                            <h3 class="card-header h5 fw-bold mb-0">
                                {$t("common.records", { n: rrs.length, type: nsrrtype(type) })}
                            </h3>
                            <div class="card-body p-0">
                                <div>
                                    <Table
                                        class="table-responsive mb-0 flush"
                                        size="sm"
                                        hover
                                        striped
                                    >
                                        <thead>
                                            <tr>
                                                {#each recordsFields(Number(type)) as field}
                                                    <th>
                                                        {$t("record." + field)}
                                                    </th>
                                                {/each}
                                                <th>
                                                    <i class="bi bi-clock"></i>
                                                    {$t("resolver.ttl")}
                                                </th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each rrs as record}
                                                <tr>
                                                    {#each recordsFields(Number(type)) as field}
                                                        <td class="font-monospace">
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
                            </div>
                        </div>
                    {/each}
                </Col>
            {/if}
        </Row>
    </Container>
{:else}
    <div class="my-5 container flex-fill">
        <div class="text-center">
            <h1 class="display-6 fw-bold">
                <i class="bi bi-search"></i>
                {$t("menu.dns-resolver")}
            </h1>
            <p class="lead mt-1">
                {$t("resolver.page-description")}
            </p>
        </div>
        <Row class="justify-content-center mt-4">
            <Col md="10" lg="8">
                <div class="card rounded-4 p-2">
                    <div class="card-body">
                        <ResolverForm bind:request_pending on:submit={resolveDomain} />
                    </div>
                </div>
            </Col>
        </Row>
    </div>
{/if}
