<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
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
    import { untrack } from "svelte";
    import { fly, fade } from "svelte/transition";
    import { cubicOut } from "svelte/easing";

    import {
        Col,
        Container,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { type DomainInfoState, createDomainInfoState, fetchDomainInfo } from "$lib/components/DomainInfoFetcher.svelte";
    import DomainInfoDisplay from "$lib/components/DomainInfoDisplay.svelte";
    import DomainInfoLookupForm from "$lib/components/DomainInfoLookupForm.svelte";
    import { t } from "$lib/translations";
    import PageTitle from "$lib/components/PageTitle.svelte";

    interface Props {
        data: { domain?: string };
    }

    let { data }: Props = $props();

    let domain = $derived(data.domain ?? "");
    let inputDomain = $state("");

    let lookup: DomainInfoState = $state(createDomainInfoState());

    $effect(() => {
        if (domain) {
            untrack(() => {
                inputDomain = domain;
                fetchDomainInfo(domain, lookup);
            });
        }
    });

    function submit() {
        if (!inputDomain) return;

        if (inputDomain === domain) {
            fetchDomainInfo(inputDomain, lookup);
        } else {
            navigate("/whois/" + encodeURIComponent(inputDomain), { noScroll: true });
        }
    }

</script>

<svelte:head>
    <title>
        {$t("domaininfo.page-title")}
        {domain ? domain : ""}
        - happyDomain
    </title>
</svelte:head>

{#if domain}
    <Container fluid class="flex-fill d-flex flex-column">
        <Row class="flex-grow-1">
            <Col md={{ offset: 0, size: 4 }} class="bg-light pt-3 pb-5">
                <div
                    class="sticky-top"
                    style="top: 1rem"
                    in:fly={{ x: -40, duration: 400, easing: cubicOut }}
                >
                    <PageTitle title={$t("domaininfo.page-title")} domain={domain} />
                    <DomainInfoLookupForm
                        bind:inputDomain
                        requestPending={lookup.pending}
                        id="domain-input"
                        onsubmit={submit}
                    />
                </div>
            </Col>

            {#if lookup.pending}
                <div class="col-md-8 pt-5 pb-5 d-flex align-items-center justify-content-center" in:fade={{ duration: 200 }}>
                    <div class="text-center text-muted">
                        <Spinner />
                        <p class="mt-3">{$t("common.spinning")}…</p>
                    </div>
                </div>
            {:else if lookup.notFound}
                <div class="col-md-8 pt-3 pb-5" in:fly={{ y: 20, duration: 350, easing: cubicOut }}>
                    <h2 class="display-7 fw-bold mt-3">
                        <i class="bi bi-question-circle"></i>
                        {domain}
                    </h2>
                    <div class="card border-warning mt-3">
                        <div class="card-body">
                            <div class="d-flex align-items-center">
                                <i class="bi bi-exclamation-triangle text-warning fs-3 me-3"></i>
                                <p class="card-text mb-0">{$t("domaininfo.domain-not-found")}</p>
                            </div>
                        </div>
                    </div>
                </div>
            {:else if lookup.error !== null}
                <div class="col-md-8 pt-3 pb-5" in:fly={{ y: 20, duration: 350, easing: cubicOut }}>
                    <h2 class="display-7 fw-bold mt-3">
                        <i class="bi bi-exclamation-triangle"></i>
                        {$t("domaininfo.error")}
                    </h2>
                    <div class="card border-danger mt-3">
                        <div class="card-body">
                            <div class="d-flex align-items-center">
                                <i class="bi bi-x-circle text-danger fs-3 me-3"></i>
                                <p class="card-text mb-0">{lookup.error}</p>
                            </div>
                        </div>
                    </div>
                </div>
            {:else if lookup.info !== null}
                <div class="col-md-8 pt-3 pb-5" in:fly={{ y: 20, duration: 400, delay: 100, easing: cubicOut }}>
                    <DomainInfoDisplay info={lookup.info} {domain} />
                </div>
            {/if}
        </Row>
    </Container>
{:else}
    <div class="my-5 container flex-fill" in:fade={{ duration: 300 }}>
        <PageTitle title={$t("domaininfo.page-title")} subtitle={$t("domaininfo.page-description")} />
        <Row class="justify-content-center mt-4">
            <Col md="10" lg="8">
                <div class="card rounded-4 p-2">
                    <div class="card-body">
                        <DomainInfoLookupForm
                            bind:inputDomain
                            requestPending={lookup.pending}
                            id="domain-input-landing"
                            onsubmit={submit}
                        />
                    </div>
                </div>
            </Col>
        </Row>
    </div>
{/if}
