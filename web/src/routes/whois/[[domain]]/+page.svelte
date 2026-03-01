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
    import { preventDefault } from "svelte/legacy";

    import {
        Button,
        Col,
        Container,
        FormGroup,
        Input,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { getDomainInfo } from "$lib/api/domaininfo";
    import type { DomainInfo } from "$lib/model/domaininfo";
    import DomainInfoDisplay from "$lib/components/DomainInfoDisplay.svelte";
    import { domains } from "$lib/stores/domains";
    import { t } from "$lib/translations";
    import PageTitle from "$lib/components/PageTitle.svelte";

    interface Props {
        data: { domain?: string };
    }

    let { data }: Props = $props();

    let domain = $derived(data.domain ?? "");
    let inputDomain = $state("");

    let error_response: string | null = $state(null);
    let not_found = $state(false);
    let request_pending = $state(false);
    let info: DomainInfo | null = $state(null);

    function fetchInfo(d: string) {
        if (!d) return;

        request_pending = true;
        error_response = null;
        not_found = false;
        info = null;

        getDomainInfo(d).then(
            (result) => {
                info = result;
                request_pending = false;
            },
            (error: unknown) => {
                const msg = error instanceof Error ? error.message : String(error);
                if (msg.toLowerCase().includes("not found") || msg.toLowerCase().includes("doesn't exist")) {
                    not_found = true;
                } else {
                    error_response = msg;
                }
                request_pending = false;
            },
        );
    }

    $effect(() => {
        if (domain) {
            untrack(() => {
                inputDomain = domain;
                fetchInfo(domain);
            });
        }
    });

    function submit() {
        if (!inputDomain) return;

        if (inputDomain === domain) {
            fetchInfo(inputDomain);
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
                <div class="sticky-top">
                    <PageTitle title={$t("domaininfo.page-title")} domain={domain} />
                    <form onsubmit={preventDefault(submit)}>
                        <FormGroup>
                            <label for="domain-input">
                                {$t("common.domain")}
                            </label>
                            <Input
                                id="domain-input"
                                class="font-monospace"
                                list="my-domains"
                                required
                                placeholder="example.com"
                                bind:value={inputDomain}
                            />
                            <div class="form-text">
                                {@html $t("domaininfo.domain-description", {
                                    domain: `<span class="font-monospace">example.com</span>`,
                                })}
                            </div>
                            <datalist id="my-domains">
                                {#if $domains}
                                    {#each $domains as dn (dn.id)}
                                        <option>{dn.domain}</option>
                                    {/each}
                                {/if}
                            </datalist>
                        </FormGroup>
                        <div class="mx-3 d-flex justify-content-end">
                            <Button type="submit" color="primary" disabled={request_pending}>
                                {#if request_pending}
                                    <Spinner size="sm" />
                                {/if}
                                {$t("domaininfo.lookup")}
                            </Button>
                        </div>
                    </form>
                </div>
            </Col>

            {#if request_pending}
                <Col md="8" class="pt-5 pb-5 d-flex align-items-center justify-content-center">
                    <div class="text-center text-muted">
                        <Spinner />
                        <p class="mt-3">{$t("common.spinning")}…</p>
                    </div>
                </Col>
            {:else if not_found}
                <Col md="8" class="pt-3 pb-5">
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
                </Col>
            {:else if error_response !== null}
                <Col md="8" class="pt-3 pb-5">
                    <h2 class="display-7 fw-bold mt-3">
                        <i class="bi bi-exclamation-triangle"></i>
                        {$t("domaininfo.error")}
                    </h2>
                    <div class="card border-danger mt-3">
                        <div class="card-body">
                            <div class="d-flex align-items-center">
                                <i class="bi bi-x-circle text-danger fs-3 me-3"></i>
                                <p class="card-text mb-0">{error_response}</p>
                            </div>
                        </div>
                    </div>
                </Col>
            {:else if info !== null}
                <Col md="8" class="pt-3 pb-5">
                    <DomainInfoDisplay {info} {domain} />
                </Col>
            {/if}
        </Row>
    </Container>
{:else}
    <div class="my-5 container flex-fill">
        <PageTitle title={$t("domaininfo.page-title")} subtitle={$t("domaininfo.page-description")} />
        <Row class="justify-content-center mt-4">
            <Col md="10" lg="8">
                <div class="card rounded-4 p-2">
                    <div class="card-body">
                        <form onsubmit={preventDefault(submit)}>
                            <FormGroup>
                                <label for="domain-input-landing">
                                    {$t("common.domain")}
                                </label>
                                <Input
                                    id="domain-input-landing"
                                    class="font-monospace"
                                    list="my-domains-landing"
                                    required
                                    placeholder="example.com"
                                    bind:value={inputDomain}
                                />
                                <div class="form-text">
                                    {@html $t("domaininfo.domain-description", {
                                        domain: `<span class="font-monospace">example.com</span>`,
                                    })}
                                </div>
                                <datalist id="my-domains-landing">
                                    {#if $domains}
                                        {#each $domains as dn (dn.id)}
                                            <option>{dn.domain}</option>
                                        {/each}
                                    {/if}
                                </datalist>
                            </FormGroup>
                            <div class="d-flex justify-content-end">
                                <Button type="submit" color="primary" disabled={request_pending}>
                                    {#if request_pending}
                                        <Spinner size="sm" />
                                    {/if}
                                    {$t("domaininfo.lookup")}
                                </Button>
                            </div>
                        </form>
                    </div>
                </div>
            </Col>
        </Row>
    </div>
{/if}
