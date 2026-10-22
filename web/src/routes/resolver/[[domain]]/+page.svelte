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
    import { goto } from "$app/navigation";
    import { resolve as resolveRoute } from "$app/paths";
    import { page } from "$app/state";
    import { untrack } from "svelte";
    import { fly, fade } from "svelte/transition";
    import { cubicOut } from "svelte/easing";

    import { Spinner, Table } from "@sveltestrap/sveltestrap";

    import { resolve as APIResolve } from "$lib/api/resolver";
    import { nsrrtype, nsttl } from "$lib/dns";
    import type { dnsRR } from "$lib/dns_rr";
    import type { ResolverForm as ResolverFormT } from "$lib/model/resolver";
    import { recordsFields } from "$lib/resolver";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    import LookupLayout, { MORPH_DURATION } from "$lib/components/LookupLayout.svelte";
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
                    responses = response.answer as Array<dnsRR>;
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
            goto(
                resolveRoute("/resolver/[[domain]]", { domain: encodeURIComponent(form.domain) }),
                {
                    state: { form, showDNSSEC },
                    noScroll: true,
                },
            );
        }
    }
</script>

<svelte:head>
    <title>
        {$t("menu.dns-resolver")}
        {domain ? domain : ""}
        - happyDomain
    </title>
</svelte:head>

<LookupLayout
    expanded={!!domain}
    title={$t("menu.dns-resolver")}
    {domain}
    subtitle={$t("resolver.page-description")}
    guide={{
        title: $t("resolver.guide.title"),
        intro: $t("resolver.guide.intro"),
        icon: "signpost-split",
        uses: [
            {
                icon: "broadcast",
                title: $t("resolver.guide.propagation.title"),
                text: $t("resolver.guide.propagation.text"),
            },
            {
                icon: "envelope-check",
                title: $t("resolver.guide.mail.title"),
                text: $t("resolver.guide.mail.text"),
            },
            {
                icon: "shield-lock",
                title: $t("resolver.guide.dnssec.title"),
                text: $t("resolver.guide.dnssec.text"),
            },
        ],
    }}
>
    {#snippet search()}
        <ResolverForm bind:request_pending bind:value={form} on:submit={resolveDomain} />
    {/snippet}

    {#if request_pending}
        <div
            class="pt-5 d-flex align-items-center justify-content-center"
            in:fade={{ duration: 200 }}
        >
            <div class="text-center text-muted">
                <Spinner />
                <p class="mt-3">{$t("wait.wait")}…</p>
            </div>
        </div>
    {:else if error_response !== null}
        <div in:fly={{ y: 20, duration: 350, easing: cubicOut }}>
            <h2 class="display-7 fw-bold mt-3">
                <i class="bi bi-exclamation-triangle"></i>
                {$t("resolver.query-failed")}
            </h2>
            <div class="card border-danger mt-3">
                <div class="card-body">
                    <div class="d-flex align-items-center">
                        <i class="bi bi-x-circle text-danger fs-3 me-3"></i>
                        <div>
                            <p class="card-text mb-1">
                                {$t("resolver.error-description")}
                            </p>
                            <code class="small">{error_response}</code>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    {:else if responses === "no-answer"}
        <div in:fly={{ y: 20, duration: 350, easing: cubicOut }}>
            <div class="card text-center py-5 mt-3">
                <div class="card-body">
                    <div class="empty-icon mb-3">
                        <i class="bi bi-inbox"></i>
                    </div>
                    <h5 class="fw-bold">
                        {$t("common.records", {
                            count: 0,
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
            filteredResponses(responses, (page.state as ResolverPageState).showDNSSEC ?? false),
        )}
        <div
            class="mt-3"
            in:fly={{ y: 20, duration: 400, delay: MORPH_DURATION * 0.6, easing: cubicOut }}
        >
            <div class="d-flex align-items-baseline gap-2 mb-1">
                <h4 class="fw-bold mb-0">
                    {$t("resolver.query-results")}
                </h4>
                <span class="badge bg-primary-subtle text-primary">
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
                    class="card mb-4 overflow-hidden"
                    in:fly={{
                        y: 30,
                        duration: 350,
                        delay: MORPH_DURATION * 0.6 + 50 + idx * 80,
                        easing: cubicOut,
                    }}
                >
                    <div class="card-header bg-body-tertiary d-flex align-items-center gap-2 py-3">
                        <span class="badge text-bg-primary">{nsrrtype(type)}</span>
                        <span class="fw-semibold">
                            {$t("common.records", {
                                count: rrs.length,
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
                                                <td class="font-monospace small ps-3">
                                                    {record[field]}
                                                </td>
                                            {/each}
                                            <td class="text-body-secondary small ps-3">
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
</LookupLayout>

<style>
    .empty-icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 4rem;
        height: 4rem;
        border-radius: 50%;
        background: var(--bs-tertiary-bg);
        font-size: 1.75rem;
        color: var(--bs-secondary-color);
    }
</style>
