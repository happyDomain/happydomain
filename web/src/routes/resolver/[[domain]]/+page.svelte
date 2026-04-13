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
    import { fly, fade } from "svelte/transition";
    import { cubicOut } from "svelte/easing";

    import { Col, Container, Row, Table } from "@sveltestrap/sveltestrap";

    import { resolve as APIResolve } from "$lib/api/resolver";
    import { nsrrtype, nsttl } from "$lib/dns";
    import type { dnsRR } from "$lib/dns_rr";
    import type { ResolverForm as ResolverFormT } from "$lib/model/resolver";
    import { recordsFields } from "$lib/resolver";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    import PageTitle from "$lib/components/PageTitle.svelte";
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
    let responses: Array<dnsRR> | "no-answer" | null = $state(null);

    function resolve(form: ResolverFormT) {
        if (!form.domain) return;

        APIResolve(form).then(
            (response) => {
                error_response = null;
                question = Object.assign({}, form);
                if (response.answer) {
                    responses = response.answer as unknown as Array<dnsRR>;
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

    function filteredResponses(responses: Array<dnsRR>, showDNSSEC: boolean): Array<dnsRR> {
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

    function responseByType(filteredResponses: Array<dnsRR>): Record<string, Array<dnsRR>> {
        const ret: Record<string, Array<dnsRR>> = {};

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
        } else if (form.domain) {
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
    <div class="resolver-layout flex-fill d-flex flex-column">
        <Container fluid class="flex-fill d-flex flex-column">
            <Row class="flex-grow-1">
                <Col md={{ offset: 0, size: 4 }} lg="3" class="resolver-sidebar pt-4 pb-5">
                    <div
                        class="sticky-top"
                        style="top: 1rem"
                        in:fly={{ x: -40, duration: 400, easing: cubicOut }}
                    >
                        <div class="sidebar-header mb-4">
                            <a
                                href="/resolver"
                                class="text-body-secondary text-decoration-none d-inline-flex align-items-center gap-1 small mb-2"
                            >
                                <i class="bi bi-arrow-left"></i>
                                {$t("menu.dns-resolver")}
                            </a>
                            <h5 class="font-monospace fw-bold text-primary mb-0">{domain}</h5>
                        </div>
                        <ResolverForm bind:request_pending value={form} on:submit={resolveDomain} />
                    </div>
                </Col>
                <Col md="8" lg="9" class="pt-4 pb-5 results-col">
                    {#if request_pending}
                        <div
                            class="d-flex flex-column align-items-center justify-content-center py-5"
                            in:fade={{ duration: 200 }}
                        >
                            <div class="resolver-loader mb-3"></div>
                            <p class="text-body-secondary">{$t("common.run")}...</p>
                        </div>
                    {:else if error_response !== null}
                        <div in:fly={{ y: 20, duration: 350, easing: cubicOut }}>
                            <div class="card border-0 shadow-sm overflow-hidden">
                                <div class="card-error-bar"></div>
                                <div class="card-body p-4">
                                    <div class="d-flex align-items-start gap-3">
                                        <div class="error-icon-wrapper">
                                            <i class="bi bi-x-circle-fill"></i>
                                        </div>
                                        <div>
                                            <h5 class="fw-bold mb-1">
                                                {$t("resolver.query-failed")}
                                            </h5>
                                            <p class="text-body-secondary mb-2">
                                                {$t("resolver.error-description")}
                                            </p>
                                            <code class="d-block bg-body-tertiary rounded p-2 small"
                                                >{error_response}</code
                                            >
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    {:else if responses === "no-answer"}
                        <div in:fly={{ y: 20, duration: 350, easing: cubicOut }}>
                            <div class="card border-0 shadow-sm text-center py-5">
                                <div class="card-body">
                                    <div class="empty-icon mb-3">
                                        <i class="bi bi-inbox"></i>
                                    </div>
                                    <h5 class="fw-bold">
                                        {$t("common.records", {
                                            n: 0,
                                            type: question ? question.type : "-",
                                        })}
                                    </h5>
                                    <p class="text-body-secondary mb-0">
                                        {$t("resolver.no-answer")}
                                    </p>
                                </div>
                            </div>
                        </div>
                    {:else if responses != null}
                        {@const resByType = responseByType(
                            filteredResponses(
                                /* @ts-ignore */ responses,
                                (page.state as ResolverPageState).showDNSSEC ?? false,
                            ),
                        )}
                        <div in:fly={{ y: 20, duration: 400, delay: 100, easing: cubicOut }}>
                            <div class="d-flex align-items-baseline gap-2 mb-1">
                                <h4 class="fw-bold mb-0">
                                    {$t("resolver.query-results")}
                                </h4>
                                <span class="badge bg-primary-subtle text-primary rounded-pill">
                                    {Object.keys(resByType).length}
                                    {Object.keys(resByType).length === 1 ? "type" : "types"}
                                </span>
                            </div>
                            <p class="text-body-secondary mb-4">
                                {#if question}
                                    {$t("resolver.results-description", {
                                        domain: question.domain,
                                        type: question.type,
                                    })}
                                {:else}
                                    {$t("resolver.results-description-default")}
                                {/if}
                            </p>
                            {#each Object.keys(resByType) as type, idx}
                                {@const rrs = resByType[type]}
                                <div
                                    class="card border-0 shadow-sm mb-4 overflow-hidden result-card"
                                    in:fly={{
                                        y: 30,
                                        duration: 350,
                                        delay: 150 + idx * 80,
                                        easing: cubicOut,
                                    }}
                                >
                                    <div
                                        class="card-header border-0 bg-body-tertiary d-flex align-items-center gap-2 py-3"
                                    >
                                        <span class="badge bg-primary rounded-pill"
                                            >{nsrrtype(type)}</span
                                        >
                                        <span class="fw-semibold">
                                            {$t("common.records", {
                                                n: rrs.length,
                                                type: nsrrtype(type),
                                            })}
                                        </span>
                                    </div>
                                    <div class="card-body p-0">
                                        <div class="table-responsive">
                                            <Table class="mb-0 align-middle" size="sm" hover>
                                                <thead>
                                                    <tr>
                                                        {#each recordsFields(Number(type)) as field}
                                                            <th
                                                                class="text-body-secondary fw-semibold small text-uppercase ps-3"
                                                            >
                                                                {$t("record." + field)}
                                                            </th>
                                                        {/each}
                                                        <th
                                                            class="text-body-secondary fw-semibold small text-uppercase ps-3"
                                                        >
                                                            <i class="bi bi-clock me-1"></i>
                                                            {$t("resolver.ttl")}
                                                        </th>
                                                    </tr>
                                                </thead>
                                                <tbody>
                                                    {#each rrs as record}
                                                        <tr>
                                                            {#each recordsFields(Number(type)) as field}
                                                                <td
                                                                    class="font-monospace small ps-3"
                                                                >
                                                                    {record[field]}
                                                                </td>
                                                            {/each}
                                                            <td
                                                                class="text-body-secondary small ps-3"
                                                            >
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
                        </div>
                    {/if}
                </Col>
            </Row>
        </Container>
    </div>
{:else}
    <div
        class="resolver-hero flex-fill d-flex flex-column align-items-center justify-content-center"
    >
        <div class="flex-fill container d-flex flex-column" in:fade={{ duration: 300 }}>
            <PageTitle title={$t("menu.dns-resolver")} subtitle={$t("resolver.page-description")} />
            <Row class="flex-fill justify-content-center align-items-center mb-5">
                <Col md="10" lg="7">
                    <div class="hero-search-card">
                        <ResolverForm bind:request_pending on:submit={resolveDomain} />
                    </div>
                </Col>
            </Row>
        </div>
    </div>
{/if}

<style>
    .resolver-hero {
        padding: 2rem 0;
    }

    .hero-search-card {
        background: #fff;
        border-radius: 1rem;
        padding: 2rem;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
    }

    .resolver-layout :global(.resolver-sidebar) {
        background: var(--bs-body-bg);
        border-right: 1px solid rgba(0, 0, 0, 0.06);
    }

    .sidebar-header {
        padding-bottom: 1rem;
        border-bottom: 1px solid rgba(0, 0, 0, 0.06);
    }

    .resolver-layout :global(.results-col) {
        background: var(--bs-tertiary-bg, #f8f9fa);
        min-height: 60vh;
    }

    .resolver-loader {
        width: 2.5rem;
        height: 2.5rem;
        border: 3px solid rgba(28, 180, 135, 0.15);
        border-top-color: var(--bs-primary);
        border-radius: 50%;
        animation: resolver-spin 0.8s linear infinite;
    }

    @keyframes resolver-spin {
        to {
            transform: rotate(360deg);
        }
    }

    .card-error-bar {
        height: 3px;
        background: linear-gradient(90deg, var(--bs-danger), transparent);
    }

    .error-icon-wrapper {
        font-size: 1.75rem;
        color: var(--bs-danger);
        line-height: 1;
        flex-shrink: 0;
    }

    .empty-icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 4rem;
        height: 4rem;
        border-radius: 50%;
        background: var(--bs-tertiary-bg, #f0f0f0);
        font-size: 1.75rem;
        color: var(--bs-secondary-color);
    }

    .result-card {
        box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
    }
</style>
