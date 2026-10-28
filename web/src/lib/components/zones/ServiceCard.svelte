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
    import {
        Dropdown,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        InputGroup,
        Progress,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { goto } from "$app/navigation";

    import ServiceBadges from "./ServiceBadges.svelte";
    import { controls as ctrlServiceDetails } from "./ServiceDetailsOffcanvas.svelte";
    import { useZoneEditor } from "./editor";
    import { deleteService, isDeletable, saveService } from "./serviceActions";
    import { getServiceSpec } from "$lib/api/service_specs";
    import DurationInput from "$lib/components/inputs/Duration.svelte";
    import { controls as ctrlRecord } from "$lib/components/modals/Record.svelte";
    import RecordText from "$lib/components/records/RecordText.svelte";
    import Service from "$lib/components/services/Service.svelte";
    import PropagationCountdown from "$lib/components/services/PropagationCountdown.svelte";
    import type { dnsRR } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceWithValue } from "$lib/model/service.svelte";
    import { serviceDescription, serviceName } from "$lib/services/infos";
    import { checkers } from "$lib/stores/checkers";
    import { checksBySubdomain } from "$lib/stores/domainchecks";
    import { isReadOnlyProvider, providers_idx } from "$lib/stores/providers";
    import { servicesSpecs, servicesSpecsLoaded } from "$lib/stores/services";
    import { t } from "$lib/translations";
    import { getStatusColor } from "$lib/utils";

    interface Props {
        dn: string;
        origin: Domain;
        service: ServiceWithValue;
        // The zone shown, the service belongs to.
        zoneId: string;
    }

    let { dn, origin, service, zoneId }: Props = $props();

    const editor = useZoneEditor(() => origin);

    let readOnly = $derived(
        editor.readOnly || isReadOnlyProvider($providers_idx, origin.id_provider),
    );
    let canDelete = $derived(!readOnly && isDeletable(service));

    let subdomain = $derived(service._domain === "@" ? "" : service._domain);
    let href = $derived(
        service._id ? editor.serviceHref(zoneId, subdomain, service._id) : undefined,
    );

    // The checks of the service; they are about the live zone.
    let checks = $derived(
        !editor.readOnly && service._id && $checksBySubdomain?.domainId === origin.id
            ? ($checksBySubdomain.sections.get(subdomain)?.items ?? []).filter(
                  (item) => item.target?.serviceId === service._id,
              )
            : [],
    );

    // Will be changed by PropagationCountdown
    let isPropagating = $state(true);

    let menuOpen = $state(false);
    let ttlSaveInProgress = $state(false);
    let deleteInProgress = $state(false);

    function saveTtl() {
        ttlSaveInProgress = true;
        saveService(editor, service).then(
            () => {
                setTimeout(() => {
                    ttlSaveInProgress = false;
                }, 500);
            },
            (err) => {
                ttlSaveInProgress = false;
                throw err;
            },
        );
    }

    function removeService() {
        if (deleteInProgress) return;
        deleteInProgress = true;
        deleteService(editor, service, zoneId).then(
            () => {
                deleteInProgress = false;
            },
            (err) => {
                deleteInProgress = false;
                throw err;
            },
        );
    }

    function editService() {
        // The editor resolves the path.
        // eslint-disable-next-line svelte/no-navigation-without-resolve
        if (!readOnly && href) goto(href);
    }
</script>

<!-- Each card stacks its own layers; with its menu open, it goes over the next ones. -->
<article class={["service-card card position-relative", menuOpen ? "z-2" : "z-1"]}>
    <header class="service-head card-body d-flex align-items-start gap-3 pb-2">
        <div class="flex-fill min-w-0">
            <div class="d-flex flex-wrap align-items-center gap-2 mb-1">
                <h3 class="h6 fw-semibold mb-0">
                    <a {href} class="stretched-link link-body-emphasis text-decoration-none">
                        {#if $servicesSpecsLoaded}
                            {$serviceName($servicesSpecs[service._svctype], service._svctype)}
                        {:else}
                            {service._svctype}
                        {/if}
                    </a>
                </h3>
                <ServiceBadges {service} />
            </div>
            {#if $servicesSpecsLoaded}
                <p class="small text-body-secondary mb-0">
                    {$serviceDescription($servicesSpecs[service._svctype], service._svctype)}
                </p>
            {/if}
            {#if service._comment}
                <p
                    class="small text-body-secondary fst-italic text-truncate mb-0"
                    title={service._comment}
                >
                    {service._comment}
                </p>
            {/if}
        </div>
        <!-- Above the link stretched over the card, and the records with their menu. -->
        <div class="d-flex align-items-center gap-2 flex-shrink-0 position-relative z-3">
            {#if service._propagated_at && isPropagating}
                <Progress
                    class="rounded flex-shrink-0"
                    style="width: 10rem"
                    barClassName="px-2 text-end"
                    value={100}
                    color="primary"
                    striped
                    animated
                >
                    <PropagationCountdown
                        bind:isPropagating
                        propagatedAt={service._propagated_at}
                        localeString="service.propagating"
                    />
                </Progress>
            {/if}
            {#if checks.length}
                <button
                    type="button"
                    class="btn btn-sm btn-light rounded-pill d-inline-flex align-items-center gap-1 border"
                    aria-label={$t("checkers.service-checks")}
                    title={$t("checkers.service-checks")}
                    onclick={() => ctrlServiceDetails.Open(service)}
                >
                    {#each checks as check (check.status.id)}
                        <span
                            class="check-dot d-inline-block p-1 rounded-circle bg-{getStatusColor(
                                check.status.latestExecution?.result?.status,
                            )}"
                            title={$checkers?.[check.status.id ?? ""]?.name ??
                                check.status.name ??
                                check.status.id}
                        ></span>
                    {/each}
                </button>
            {/if}
            <Dropdown size="sm" autoClose="outside" bind:isOpen={menuOpen}>
                <DropdownToggle
                    color="secondary"
                    outline
                    size="sm"
                    title={$t("service.actions")}
                    aria-label={$t("service.actions")}
                >
                    ···
                </DropdownToggle>
                <DropdownMenu end style="width: 16rem">
                    {#if service._id}
                        <div class="px-3 py-1">
                            <label
                                for={`svc_ttl_${service._id}`}
                                title={$t("service.ttl-long")}
                                class="form-label d-flex align-items-center gap-2 small text-body-secondary mb-1"
                            >
                                {$t("service.default-ttl")}
                                {#if ttlSaveInProgress}
                                    <Spinner size="sm" />
                                {/if}
                            </label>
                            <InputGroup size="sm">
                                <DurationInput
                                    id={`svc_ttl_${service._id}`}
                                    title={readOnly
                                        ? $t("domains.read-only-edit")
                                        : $t("service.ttl-tip")}
                                    disabled={readOnly}
                                    zeroHint={$t("service.ttl-zero")}
                                    bind:value={service._ttl}
                                    onchange={saveTtl}
                                />
                            </InputGroup>
                        </div>
                        <DropdownItem divider />
                    {/if}
                    <DropdownItem
                        disabled={readOnly || !href}
                        aria-disabled={readOnly || !href}
                        title={readOnly ? $t("domains.read-only-edit") : undefined}
                        on:click={editService}
                    >
                        <Icon name="pencil" />
                        {$t("service.edit")}
                    </DropdownItem>
                    {#if canDelete}
                        <DropdownItem
                            class="text-danger"
                            disabled={deleteInProgress}
                            on:click={removeService}
                        >
                            {#if deleteInProgress}
                                <Spinner size="sm" />
                            {:else}
                                <Icon name="trash" />
                            {/if}
                            {$t("service.delete")}
                        </DropdownItem>
                    {/if}
                </DropdownMenu>
            </Dropdown>
        </div>
    </header>

    <div class="service-records list-group list-group-flush mx-3 mb-3 pt-2 border-top">
        {#await getServiceSpec(service._svctype)}
            <div class="text-center text-body-tertiary small py-1">
                <Spinner size="sm" class="me-1" />
                {$t("common.loading")}
            </div>
        {:then specs}
            <Service {specs} bind:value={service.Service}>
                {#snippet aservice(_type: string, rr: unknown)}
                    {#if rr}
                        <!-- Above the link stretched over the card. -->
                        <button
                            type="button"
                            class="list-group-item list-group-item-action position-relative z-2 px-2 py-1 small"
                            onclick={() => ctrlRecord.Open(rr as dnsRR, service._domain)}
                        >
                            <RecordText {dn} {origin} rr={rr as dnsRR} />
                        </button>
                    {/if}
                {/snippet}
            </Service>
        {/await}
    </div>
</article>

<style>
    /* Only the service is tinted on hover, so the records keep their own hover. */
    .service-head {
        border-top-left-radius: var(--bs-card-inner-border-radius);
        border-top-right-radius: var(--bs-card-inner-border-radius);
        transition: background-color 0.15s;
    }

    .service-card:hover .service-head {
        background: var(--hd-bg-hover);
    }

    .service-records {
        --bs-border-style: dashed;
        --bs-list-group-bg: transparent;
    }

    .service-records:empty {
        display: none;
    }
</style>
