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
    import { Col, Row } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import { SERVICE_FAMILY_HIDDEN } from "$lib/model/service_specs.svelte";
    import { servicesSpecs } from "$lib/services_specs";
    import type { ServiceInfos } from "$lib/model/service_specs.svelte";
    import { t } from "$lib/translations";

    function groupByCategory(specs: Record<string, ServiceInfos>): Record<string, ServiceInfos[]> {
        const groups: Record<string, ServiceInfos[]> = {};
        for (const svc of Object.values(specs)) {
            if (svc.family === SERVICE_FAMILY_HIDDEN) continue;
            const cats = svc.categories && svc.categories.length > 0 ? svc.categories : ["general"];
            for (const cat of cats) {
                if (!groups[cat]) groups[cat] = [];
                groups[cat].push(svc);
            }
        }
        return groups;
    }

    let grouped = $derived(groupByCategory(servicesSpecs));
    let categoryNames = $derived(Object.keys(grouped).sort());
</script>

<svelte:head>
    <title>{$t("generator.title")} - happyDomain</title>
    <meta name="description" content={$t("generator.description")} />
</svelte:head>

<div class="my-5 container flex-fill">
    <PageTitle title={$t("generator.title")} subtitle={$t("generator.subtitle")} />

    {#each categoryNames as category}
        <section class="mb-5">
            <h2 class="h4 text-capitalize mb-3">{category}</h2>
            <Row cols={{ xs: 1, sm: 2, md: 3, lg: 4 }}>
                {#each grouped[category] as svc (svc._svctype)}
                    <Col class="mb-3">
                        <a
                            href="/generator/{encodeURIComponent(svc._svctype)}"
                            class="card h-100 text-decoration-none text-reset generator-card"
                        >
                            <div class="card-body d-flex align-items-start gap-3">
                                {#if svc._svcicon}
                                    <img
                                        src="/api/service_specs/{encodeURIComponent(
                                            svc._svctype,
                                        )}/icon.png"
                                        alt={$t("generator.icon-alt", { name: svc.name })}
                                        width="32"
                                        height="32"
                                        class="flex-shrink-0"
                                        style="object-fit: contain;"
                                    />
                                {:else}
                                    <span class="flex-shrink-0 fs-4">📄</span>
                                {/if}
                                <div>
                                    <h3 class="h6 card-title mb-1">{svc.name}</h3>
                                    {#if svc.description}
                                        <p class="card-text small text-muted mb-0">
                                            {svc.description}
                                        </p>
                                    {/if}
                                </div>
                            </div>
                        </a>
                    </Col>
                {/each}
            </Row>
        </section>
    {/each}
</div>

<style>
    .generator-card {
        border: 1px solid rgba(0, 0, 0, 0.1);
        transition:
            box-shadow 0.15s ease,
            border-color 0.15s ease;
    }
    .generator-card:hover {
        box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
        border-color: var(--bs-primary);
    }
</style>
