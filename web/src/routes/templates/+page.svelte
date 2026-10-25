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
    import { onMount, tick } from "svelte";
    import { fade } from "svelte/transition";

    import { Button, Card, CardBody, Col, Container, Row, Spinner } from "@sveltestrap/sveltestrap";

    import { deleteZoneTemplate, updateZoneTemplate } from "$lib/api/zone_template";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import ModalTemplateDetails, {
        controls as ctrlTemplateDetails,
    } from "$lib/components/zone_templates/ModalTemplateDetails.svelte";
    import TemplateTable from "$lib/components/zone_templates/TemplateTable.svelte";
    import {
        isGlobalTemplate,
        type ZoneTemplate,
        type ZoneTemplateInput,
    } from "$lib/model/zone_template";
    import { toasts } from "$lib/stores/toasts";
    import {
        refreshZoneTemplates,
        removeZoneTemplate,
        upsertZoneTemplate,
        zoneTemplates,
    } from "$lib/stores/zone_templates";
    import { t } from "$lib/translations";

    let loadError = $state<string | null>(null);

    let mine = $derived($zoneTemplates?.filter((tpl) => !isGlobalTemplate(tpl)) ?? []);

    function load() {
        // The store may already hold the templates, loaded by another page:
        // they are shown at once, while refreshed.
        refreshZoneTemplates().then(
            () => (loadError = null),
            (err) => {
                loadError = String(err);
                // The list shown may be outdated: tell it.
                if ($zoneTemplates !== undefined) {
                    toasts.addErrorToast({
                        title: $t("zone-templates.errors.load"),
                        message: String(err),
                        timeout: 8000,
                    });
                }
            },
        );
    }

    onMount(load);

    let editing = $state<ZoneTemplate | null>(null);

    async function edit(tpl: ZoneTemplate) {
        editing = tpl;
        // The modal reads the template when it opens: let it be mounted.
        await tick();
        ctrlTemplateDetails.Open();
    }

    async function saveDetails(input: ZoneTemplateInput): Promise<ZoneTemplate> {
        const tpl = await updateZoneTemplate(editing!.id, input);
        upsertZoneTemplate(tpl);
        return tpl;
    }

    async function remove(tpl: ZoneTemplate) {
        if (!confirm($t("zone-templates.delete-confirm", { name: tpl.name }))) return;

        try {
            await deleteZoneTemplate(tpl.id);
            removeZoneTemplate(tpl.id);
            toasts.addToast({
                type: "success",
                title: $t("zone-templates.deleted", { name: tpl.name }),
                timeout: 5000,
            });
        } catch (err) {
            toasts.addErrorToast({
                title: $t("zone-templates.errors.delete"),
                message: err instanceof Error ? err.message : String(err),
                timeout: 8000,
            });
        }
    }
</script>

<svelte:head>
    <title>{$t("zone-templates.title")} - happyDomain</title>
</svelte:head>

<Container class="flex-fill mt-4 mb-5">
    <PageTitle title={$t("zone-templates.title")} subtitle={$t("zone-templates.description")} />

    {#if loadError && $zoneTemplates === undefined}
        <Card body color="danger" outline>
            <p class="mb-2">{$t("zone-templates.errors.load")}: {loadError}</p>
            <Button size="sm" color="danger" outline on:click={load}>
                {$t("zone-templates.retry")}
            </Button>
        </Card>
    {:else if $zoneTemplates === undefined}
        <Card body>
            <p class="text-center mb-0">
                <Spinner size="sm" class="me-2" />
                {$t("zone-templates.loading")}
            </p>
        </Card>
    {:else}
        {#if mine.length === 0}
            <div in:fade>
                <Card class="empty-state border-0 bg-body-tertiary mb-4">
                    <CardBody class="p-4 p-md-5">
                        <h2 class="h4 mb-3">
                            <i class="bi bi-files text-primary me-2"></i>{$t(
                                "zone-templates.empty.title",
                            )}
                        </h2>
                        <p class="text-muted">{$t("zone-templates.empty.intro")}</p>
                        <Row class="g-4 mt-1">
                            {#each ["save", "edit", "apply"] as step, i (step)}
                                <Col md={4}>
                                    <div class="d-flex gap-3">
                                        <span class="step-number">{i + 1}</span>
                                        <div>
                                            <h3 class="h6 fw-bold mb-1">
                                                {$t(`zone-templates.empty.${step}.title`)}
                                            </h3>
                                            <p class="small text-muted mb-0">
                                                {$t(`zone-templates.empty.${step}.text`)}
                                            </p>
                                        </div>
                                    </div>
                                </Col>
                            {/each}
                        </Row>
                    </CardBody>
                </Card>
            </div>
        {/if}

        {#if $zoneTemplates.length > 0}
            <TemplateTable items={$zoneTemplates} onedit={edit} ondelete={remove} />
        {/if}
    {/if}
</Container>

{#if editing}
    <ModalTemplateDetails template={editing} save={saveDetails} />
{/if}

<style>
    .step-number {
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 2rem;
        height: 2rem;
        border-radius: 50%;
        background: var(--bs-primary);
        color: #fff;
        font-weight: 700;
    }
</style>
