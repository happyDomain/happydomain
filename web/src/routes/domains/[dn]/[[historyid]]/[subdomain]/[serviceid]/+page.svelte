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
    // @ts-ignore
    import { escape } from "html-escaper";
    import { onDestroy } from "svelte";
    import { page } from "$app/state";

    import { Button, Icon, Input, Label, Spinner } from "@sveltestrap/sveltestrap";

    import { initializeService } from "$lib/api/service_specs";
    import { addZoneService, deleteZoneService, updateZoneService } from "$lib/api/zone";
    import ServiceEditor from "$lib/components/services/ServiceEditor.svelte";
    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { ServiceCombined } from "$lib/model/service.svelte";
    import { helpLinkOverride } from "$lib/stores/help";
    import { servicesSpecs, servicesSpecsLoaded } from "$lib/stores/services";
    import { thisZone } from "$lib/stores/thiszone";
    import { navigate } from "$lib/stores/config";
    import { t } from "$lib/translations";

    interface Props {
        data: {
            domain: Domain;
            history: string;
            zoneId: string;
            subdomain: string;
            serviceid: string;
        };
    }

    let { data }: Props = $props();

    let svcType: string = $derived(page.url.searchParams.get("type") ?? "");

    let service: ServiceCombined | null = $state(null);
    let serviceLoading = $state(false);

    $effect(() => {
        if (data.serviceid !== "new") {
            const svcs = $thisZone?.services[data.subdomain];
            service = svcs?.find((s) => s._id === data.serviceid) ?? null;
        }
    });

    $effect(() => {
        if (data.serviceid === "new" && svcType) {
            serviceLoading = true;
            initializeService(svcType).then((svc) => {
                service = new ServiceCombined({
                    _svctype: svcType,
                    _domain: data.subdomain,
                    Service: svc,
                });
                serviceLoading = false;
            });
        }
    });

    let addServiceInProgress = $state(false);
    let deleteServiceInProgress = $state(false);

    function goBack() {
        navigate(
            `/domains/${encodeURIComponent(data.domain.domain)}/${encodeURIComponent(data.history)}`,
        );
    }

    function deleteService() {
        if (!service || !$thisZone) return;

        deleteServiceInProgress = true;
        deleteZoneService(data.domain, $thisZone.id, service).then(
            (z) => {
                thisZone.set(z);
                deleteServiceInProgress = false;
                goBack();
            },
            (err) => {
                deleteServiceInProgress = false;
                throw err;
            },
        );
    }

    function submitServiceForm(e: SubmitEvent) {
        e.preventDefault();
        if (!service || !$thisZone) return;

        addServiceInProgress = true;
        const action = service._id ? updateZoneService : addZoneService;

        action(data.domain, $thisZone.id, service).then(
            (z) => {
                thisZone.set(z);
                addServiceInProgress = false;
                goBack();
            },
            (err) => {
                addServiceInProgress = false;
                throw err;
            },
        );
    }

    function helpLink(svc: ServiceCombined | null): string {
        if (!svc?._svctype) return "";
        const svcPart = svc._svctype.toLowerCase().split(".");
        let path = svcPart[svcPart.length - 1] + "/";
        if (svcPart.length === 2) {
            if (svcPart[0] === "svcs") path = "records/" + svcPart[1].toUpperCase() + "/";
            else if (svcPart[0] === "abstract") path = "services/" + svcPart[1] + "/";
            else if (svcPart[0] === "provider") path = "services/providers/" + svcPart[1] + "/";
        }
        return "reference/" + path;
    }

    onDestroy(() => helpLinkOverride.set(null));

    $effect(() => {
        helpLinkOverride.set(helpLink(service));
    });

    let canDelete = $derived(
        !!service?._id &&
            service._svctype !== "abstract.Origin" &&
            service._svctype !== "abstract.NSOnlyOrigin",
    );
</script>

{#if serviceLoading || (data.serviceid !== "new" && !$thisZone)}
    <div class="d-flex justify-content-center mt-4">
        <Spinner />
    </div>
{:else if service}
    <div class="flex-fill">
        <h2 class="d-flex align-items-center gap-2 pt-2 rounded">
            <Button
                color="link"
                class="p-0 text-reset"
                title={$t("common.cancel")}
                on:click={goBack}
            >
                <Icon name="chevron-left" />
            </Button>
            {#if service._id}
                {#if $servicesSpecsLoaded && $servicesSpecs[service._svctype]}
                    {$t("common.update-what", {
                        what: $servicesSpecs[service._svctype].name,
                    } as any)}
                {:else}
                    {$t("service.update")}
                {/if}
            {:else}
                {@html $t("service.form-new", {
                    domain: `<span class="font-monospace">${escape(fqdn(service._domain, data.domain.domain))}</span>`,
                })}
            {/if}
        </h2>

        <form id="addSvcForm" class="mt-2" onsubmit={submitServiceForm}>
            {#if !$servicesSpecsLoaded}
                <div class="d-flex justify-content-center">
                    <Spinner />
                </div>
            {:else}
                <ServiceEditor
                    dn={service._domain}
                    origin={data.domain}
                    type={service._svctype}
                    bind:value={service.Service}
                />
            {/if}
        </form>

        <div class="d-flex justify-content-end align-items-center gap-2 mt-3">
            <Label for="svc_ttl" title={$t("service.ttl-long")}>{$t("service.ttl")}</Label>
            <Input
                id="svc_ttl"
                min="0"
                type="number"
                style="width: 8em"
                title={$t("service.ttl-tip")}
                bind:value={service._ttl}
                on:input={(e: any) =>
                    parseInt(e.target.value, 10)
                        ? (service!._ttl = parseInt(e.target.value, 10))
                        : (service!._ttl = 0)}
            />
            {#if canDelete}
                <Button
                    color="danger"
                    disabled={addServiceInProgress || deleteServiceInProgress}
                    title={$t("service.delete")}
                    on:click={deleteService}
                >
                    {#if deleteServiceInProgress}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="trash" />
                    {/if}
                </Button>
            {/if}
            {#if service._id}
                <Button
                    disabled={addServiceInProgress || deleteServiceInProgress}
                    form="addSvcForm"
                    type="submit"
                    color="success"
                >
                    {#if addServiceInProgress}
                        <Spinner size="sm" />
                    {/if}
                    {$t("service.update")}
                </Button>
            {:else}
                <Button form="addSvcForm" type="submit" color="primary">
                    {$t("service.add")}
                </Button>
            {/if}
        </div>
    </div>
{:else}
    <div class="alert alert-warning m-3">
        {$t("errors.404.content")}
    </div>
{/if}
