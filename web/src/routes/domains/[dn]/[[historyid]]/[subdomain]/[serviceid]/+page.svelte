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
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { onDestroy } from "svelte";
    import { page } from "$app/state";

    import { Button, Icon, Spinner } from "@sveltestrap/sveltestrap";

    import type { ServiceWithValue } from "$lib/model/service.svelte";
    import { initializeService } from "$lib/api/service_specs";
    import { addZoneService, updateZoneService } from "$lib/api/zone";
    import {
        newServiceIdIn,
        notifyServiceChange,
        zoneWasForked,
    } from "$lib/stores/zonefeedback";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import ServiceEditor from "$lib/components/services/ServiceEditor.svelte";
    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { ServiceCombined } from "$lib/model/service.svelte";
    import { domainLink } from "$lib/stores/domains";
    import { helpLinkOverride } from "$lib/stores/help";
    import { serviceDescription, serviceName } from "$lib/services/infos";
    import { servicesSpecs, servicesSpecsLoaded } from "$lib/stores/services";
    import { thisZone } from "$lib/stores/thiszone";
        import { t } from "$lib/translations";
    import { refreshDomains } from "$lib/stores/domains";

    interface Props {
        data: {
            domain: Domain;
            history: string;
            zoneId: string;
            subdomain: string;
            serviceid: string;
        };
    }

    // The shape is what load() returns; a page reading part of it is not a
    // reason to describe it any less accurately.
    // eslint-disable-next-line svelte/no-unused-props
    let { data }: Props = $props();

    let svcType: string = $derived(page.url.searchParams.get("type") ?? "");

    let service: ServiceWithValue | undefined = $state();
    let serviceLoading = $state(false);

    let serviceTitle = $derived.by(() => {
        const svc = service;
        if (!svc) return "";
        if (svc._id) {
            return $servicesSpecsLoaded && $servicesSpecs[svc._svctype]
                ? $t("common.update-what", {
                      what: $serviceName($servicesSpecs[svc._svctype], svc._svctype),
                  } as Record<string, string>)
                : $t("service.update");
        }
        return $t("service.add");
    });

    $effect(() => {
        if (data.serviceid !== "new") {
            const svcs = $thisZone?.services[data.subdomain];
            service = svcs?.find((s) => s._id === data.serviceid);
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

    function goBack(historyid?: string) {
        goto(
            resolve("/domains/[dn]/[[historyid]]", {
                dn: encodeURIComponent(domainLink(data.domain.id)),
                historyid: encodeURIComponent(historyid ? historyid : data.history),
            }),
        );
    }

    function submitServiceForm(e: SubmitEvent) {
        e.preventDefault();
        if (!service || !$thisZone || addServiceInProgress) return;

        addServiceInProgress = true;
        const isNew = !service._id;
        const action = service._id ? updateZoneService : addZoneService;
        // The zone is replaced below, but telling what changed needs the one we
        // started from.
        const previousZone = $thisZone;
        const svctype = service._svctype;
        const dn = service._domain;

        action(data.domain, $thisZone.id, service).then(
            (z) => {
                thisZone.set(z);
                // Keep the button busy until the user is actually back on the
                // zone: the domains refresh and the navigation are part of the
                // wait, and releasing it here would let a second submit through.
                const serviceId = isNew ? newServiceIdIn(previousZone, z, dn) : service?._id;
                const forked = zoneWasForked(previousZone, z);
                refreshDomains().then(() => {
                    // Confirm as we hand the user back to the zone, not before:
                    // the highlight only lasts a couple of seconds, and it has
                    // to still be running once the grid is on screen.
                    notifyServiceChange(isNew ? "added" : "updated", {
                        svctype,
                        dn,
                        serviceId,
                        // Coming back lands at the top of the grid, so the card
                        // has to be brought into view for the highlight to be
                        // seen at all, whether it is a new one or not.
                        scroll: true,
                        forked,
                    });
                    goBack(z.id);
                });
            },
            (err) => {
                addServiceInProgress = false;
                throw err;
            },
        );
    }

    function helpLink(svc: ServiceWithValue | undefined): string {
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
</script>

{#if serviceLoading || (data.serviceid !== "new" && !$thisZone)}
    <div class="d-flex justify-content-center mt-4">
        <Spinner />
    </div>
{:else if service}
    <div class="flex-fill">
        <PageTitle
            title={serviceTitle}
            subtitle={$serviceDescription($servicesSpecs[service._svctype], service._svctype) ||
                undefined}
            domain={fqdn(service._domain, data.domain.domain)}
        />

        <form id="addSvcForm" onsubmit={submitServiceForm}>
            {#if !$servicesSpecsLoaded}
                <div class="d-flex justify-content-center">
                    <Spinner />
                </div>
            {:else}
                {#key data.serviceid}
                    <ServiceEditor
                        dn={service._domain}
                        origin={data.domain}
                        type={service._svctype}
                        bind:value={service.Service}
                    />
                {/key}
            {/if}
        </form>

        <div class="d-flex justify-content-end align-items-center gap-2 mt-3">
            {#if service._id}
                <Button
                    color="info"
                    outline
                    href="checks"
                >
                    <Icon name="shield-check" />
                    {$t("checkers.title")}
                </Button>
                <Button
                    disabled={addServiceInProgress}
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
                <Button
                    disabled={addServiceInProgress}
                    form="addSvcForm"
                    type="submit"
                    color="primary"
                >
                    {#if addServiceInProgress}
                        <Spinner size="sm" />
                    {/if}
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
