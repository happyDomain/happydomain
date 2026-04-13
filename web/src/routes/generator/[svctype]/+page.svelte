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

    import { generateServiceRecords } from "$lib/api/service_specs";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import ServiceEditor from "$lib/components/services/ServiceEditor.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceInfos } from "$lib/model/service_specs.svelte";
    import { t } from "$lib/translations";
    import { printRR } from "$lib/dns";
    import type { dnsRR } from "$lib/dns_rr";

    interface Props {
        data: { svctype: string; spec: ServiceInfos };
    }

    let { data }: Props = $props();

    let svctype = $derived(data.svctype);
    let svcSpec = $derived(data.spec);

    let domain: string = $state("");
    let serviceValue: Record<string, unknown> = $state({});

    let recordsPromise: Promise<dnsRR[]> | null = $state(null);
    let generateDebounce: ReturnType<typeof setTimeout>;

    $effect(() => {
        JSON.stringify(serviceValue); // track all changes
        domain; // track domain changes
        clearTimeout(generateDebounce);
        generateDebounce = setTimeout(() => {
            recordsPromise = generateServiceRecords(
                svctype,
                serviceValue,
                (domain && !domain.endsWith(".") ? domain + "." : domain) || undefined,
            );
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
    {#if svcSpec}
        <title>{$t("generator.svctype.title", { name: svcSpec.name })} - happyDomain</title>
        <meta
            name="description"
            content={$t("generator.svctype.description", { name: svcSpec.name })}
        />
    {:else}
        <title>{$t("generator.svctype.page-title")} - happyDomain</title>
    {/if}
</svelte:head>

<Container fluid class="my-4 flex-fill">
    <Row class="justify-content-center">
        <Col lg="8" xl="7">
            <div class="mb-3">
                <a href="/generator" class="text-body-secondary text-decoration-none small">
                    <i class="bi bi-arrow-left me-1"></i>
                    {$t("common.back")}
                </a>
            </div>

            <PageTitle
                title={$t("generator.svctype.title", { name: svcSpec.name })}
                subtitle={svcSpec.description}
            />

            <div class="step-card mb-4">
                <div class="step-header">
                    <span class="step-number">1</span>
                    <h4 class="step-title">{$t("generator.svctype.domain-settings")}</h4>
                </div>
                <div class="step-body">
                    <p class="text-body-secondary small mb-2">
                        {$t("generator.svctype.domain-help")}
                    </p>
                    <Input type="text" autofocus placeholder="example.com." bind:value={domain} />
                </div>
            </div>

            <div class="step-card mb-4">
                <div class="step-header">
                    <span class="step-number">2</span>
                    <h4 class="step-title">{$t("generator.svctype.configure-record")}</h4>
                </div>
                <div class="step-body">
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

            <div class="step-card mb-4">
                <div class="step-header">
                    <span class="step-number">3</span>
                    <h4 class="step-title">{$t("generator.svctype.generated-records")}</h4>
                </div>
                <div class="step-body p-0">
                    {#if recordsPromise === null}
                        <div class="p-3 text-body-secondary small">
                            {$t("generator.svctype.fill-form")}
                        </div>
                    {:else}
                        {#await recordsPromise}
                            <div class="p-3 d-flex align-items-center gap-2 text-body-secondary">
                                <Spinner size="sm" />
                                <span>{$t("generator.svctype.generating")}</span>
                            </div>
                        {:then records}
                            {#if records && records.length > 0}
                                <pre class="records-output">{records
                                        .map((rr) => printRR(rr))
                                        .join("\n")}</pre>
                            {:else}
                                <div class="p-3 text-body-secondary small">
                                    {$t("generator.svctype.no-records")}
                                </div>
                            {/if}
                        {:catch err}
                            <div class="p-3 text-danger small">{err.message}</div>
                        {/await}
                    {/if}
                </div>
            </div>

            <div class="cta-card mb-4">
                <div class="d-flex flex-column flex-sm-row align-items-start align-items-sm-center gap-3">
                    <div class="flex-grow-1">
                        <h5 class="fw-bold mb-1">{$t("generator.svctype.cta-title")}</h5>
                        <p class="text-body-secondary small mb-0">
                            {$t("generator.svctype.cta-text")}
                        </p>
                    </div>
                    <a href="/register" class="btn btn-primary flex-shrink-0">
                        {$t("generator.svctype.cta-button")}
                    </a>
                </div>
            </div>
        </Col>
    </Row>
</Container>

<style>
    .step-card {
        background: #fff;
        border: 1px solid rgba(0, 0, 0, 0.08);
        border-radius: 0.75rem;
        overflow: hidden;
        box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
    }

    .step-header {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        padding: 1rem 1.25rem;
        border-bottom: 1px solid rgba(0, 0, 0, 0.06);
        background: rgba(0, 0, 0, 0.015);
    }

    .step-number {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 1.75rem;
        height: 1.75rem;
        border-radius: 50%;
        background: var(--bs-primary);
        color: #fff;
        font-size: 0.8rem;
        font-weight: 700;
        flex-shrink: 0;
    }

    .step-title {
        font-size: 1rem;
        font-weight: 600;
        margin: 0;
    }

    .step-body {
        padding: 1.25rem;
    }

    .records-output {
        margin: 0;
        padding: 1rem 1.25rem;
        font-size: 0.8rem;
        background: #f8f9fa;
        border-top: 1px solid rgba(0, 0, 0, 0.04);
        overflow-x: auto;
    }

    .cta-card {
        background: linear-gradient(135deg, #f0faf7 0%, #e8f4f8 100%);
        border: 1px solid rgba(28, 180, 135, 0.2);
        border-radius: 0.75rem;
        padding: 1.5rem;
    }
</style>
