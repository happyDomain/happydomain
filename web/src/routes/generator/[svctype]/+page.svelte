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
    import { Col, Container, Input, Row, Spinner } from "@sveltestrap/sveltestrap";

    import {
        generateServiceRecords,
        initializeService,
        listServiceSpecs,
    } from "$lib/api/service_specs";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import ServiceEditor from "$lib/components/services/ServiceEditor.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceInfos } from "$lib/model/service_specs.svelte";
    import { t } from "$lib/translations";
    import { printRR } from "$lib/dns";
    import type { dnsRR } from "$lib/dns_rr";

    interface Props {
        data: { svctype: string };
    }

    let { data }: Props = $props();

    let svctype = $derived(data.svctype);

    let dataPromise = $derived(Promise.all([listServiceSpecs(), initializeService(svctype)]));

    let spec: ServiceInfos | null = $state(null);

    $effect(() => {
        dataPromise
            .then(([specs, _iv]) => {
                spec = specs[svctype] ?? null;
            })
            .catch(() => {
                spec = null;
            });
    });

    let domain: string = $state("");

    let serviceValue: any = $state({});

    $effect(() => {
        dataPromise
            .then(([_, iv]) => {
                serviceValue = iv ?? {};
            })
            .catch(() => {
                serviceValue = {};
            });
    });

    let recordsPromise: Promise<dnsRR[]> | null = $state(null);
    let generateDebounce: ReturnType<typeof setTimeout>;

    $effect(() => {
        JSON.stringify(serviceValue); // track all changes
        domain; // track domain changes
        clearTimeout(generateDebounce);
        generateDebounce = setTimeout(() => {
            recordsPromise = generateServiceRecords(svctype, serviceValue, (domain && !domain.endsWith(".") ? domain + "." : domain) || undefined);
        }, 400);
    });

    let mockDomain: Domain = $derived({
        id: "preview",
        id_provider: "preview",
        domain: domain || "example.com.",
        id_owner: "preview",
        group: "",
        zone_history: [],
    });
</script>

<svelte:head>
    {#if spec}
        <title>{$t("generator.svctype.title", { name: spec.name })} - happyDomain</title>
        <meta
            name="description"
            content={$t("generator.svctype.description", { name: spec.name })}
        />
    {:else}
        <title>{$t("generator.svctype.page-title")} - happyDomain</title>
    {/if}
</svelte:head>

{#await dataPromise}
    <div class="d-flex justify-content-center mt-5">
        <Spinner />
    </div>
{:then [specs, _iv]}
    {#if !specs[svctype]}
        <div class="my-5 container flex-fill">
            <div class="alert alert-warning">
                {@html $t("generator.svctype.not-found", { svctype })}
                <a href="/generator">{$t("generator.svctype.browse-all")}</a>
            </div>
        </div>
    {:else}
        {@const svcSpec = specs[svctype]}

        <Container fluid class="my-4 flex-fill">
            <Row class="justify-content-center">
                <Col lg="8" xl="7">
                    <PageTitle
                        title={$t("generator.svctype.title", { name: svcSpec.name })}
                        subtitle={svcSpec.description}
                    />

                    <div class="card mb-4">
                        <h4 class="card-header fw-semibold">
                            1. {$t("generator.svctype.domain-settings")}
                        </h4>
                        <div class="card-body">
                            <p class="text-muted small mb-2">
                                {$t("generator.svctype.domain-help")}
                            </p>
                            <Input type="text" autofocus placeholder="example.com." bind:value={domain} />
                        </div>
                    </div>

                    <div class="card mb-4">
                        <h4 class="card-header fw-semibold">
                            2. {$t("generator.svctype.configure-record")}
                        </h4>
                        <div class="card-body">
                            {#key svctype}
                                <ServiceEditor
                                    dn=""
                                    origin={mockDomain}
                                    type={svctype}
                                    bind:value={serviceValue}
                                />
                            {/key}
                        </div>
                    </div>

                    <div class="card mb-4">
                        <h4 class="card-header fw-semibold">
                            3. {$t("generator.svctype.generated-records")}
                        </h4>
                        <div class="card-body p-0">
                            {#if recordsPromise === null}
                                <div class="p-3 text-muted small">
                                    {$t("generator.svctype.fill-form")}
                                </div>
                            {:else}
                                {#await recordsPromise}
                                    <div class="p-3 d-flex align-items-center gap-2 text-muted">
                                        <Spinner size="sm" />
                                        <span>{$t("generator.svctype.generating")}</span>
                                    </div>
                                {:then records}
                                    {#if records && records.length > 0}
                                        <pre class="mb-0 p-3 font-monospace small">{records.map((rr) => printRR(rr)).join(
                                                "\n",
                                            )}</pre>
                                    {:else}
                                        <div class="p-3 text-muted small">
                                            {$t("generator.svctype.no-records")}
                                        </div>
                                    {/if}
                                {:catch err}
                                    <div class="p-3 text-danger small">{err.message}</div>
                                {/await}
                            {/if}
                        </div>
                    </div>

                    <div class="card border-primary mb-4">
                        <div class="card-body">
                            <h5 class="card-title">{$t("generator.svctype.cta-title")}</h5>
                            <p class="card-text text-muted">
                                {$t("generator.svctype.cta-text")}
                            </p>
                            <a href="/join" class="btn btn-primary">{$t("generator.svctype.cta-button")}</a>
                        </div>
                    </div>
                </Col>
            </Row>
        </Container>
    {/if}
{:catch err}
    <div class="my-5 container flex-fill">
        <div class="alert alert-danger">{err.message}</div>
    </div>
{/await}
