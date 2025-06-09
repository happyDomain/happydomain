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
    // @ts-ignore
    import { escape } from "html-escaper";
    import {
        Badge,
        Button,
        Card,
        CardHeader,
        Icon,
        ListGroup,
        ListGroupItem,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { addDomain } from "$lib/api/domains";
    import { createDomain, listImportableDomains } from "$lib/api/provider";
    import DomainWithProvider from "$lib/components/domains/DomainWithProvider.svelte";
    import { fqdn, fqdnCompare, validateDomain } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { Provider } from "$lib/model/provider";
    import { filteredName } from '$lib/stores/home';
    import { providersSpecs } from "$lib/stores/providers";
    import { domains_idx, refreshDomains } from "$lib/stores/domains";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    export let provider: Provider;

    let importableDomainsList: Array<string> | null = null;
    let discoveryError: string | null = null;
    export let noDomainsList = false;

    $: {
        importableDomainsList = null;
        discoveryError = null;
        noDomainsList = false;
        listImportableDomains(provider).then(
            (l) => {
                if (l === null) {
                    importableDomainsList = [];
                } else {
                    l.sort(fqdnCompare);
                    importableDomainsList = l;
                }
            },
            (err) => {
                importableDomainsList = [];
                if (err.name == "ProviderNoDomainListingSupport") {
                    noDomainsList = true;
                } else {
                    discoveryError = err.message;
                    throw err;
                }
            },
        );
    }

    function haveDomain($domains_idx: Record<string, Domain>, name: string) {
        let domain: Domain | undefined = undefined;
        if (name[name.length - 1] == ".") {
            domain = $domains_idx[name];
        } else {
            domain = $domains_idx[name + "."];
        }
        return domain !== undefined && domain.id_provider == provider._id;
    }

    async function importDomain(domain: { domain: string; wait: boolean }, noToast: bool) {
        domain.wait = true;
        addDomain(domain.domain, provider).then(
            (mydomain) => {
                domain.wait = false;
                if (!noToast) {
                    toasts.addToast({
                        title: $t("domains.attached-new"),
                        message: $t("domains.added-success", { domain: mydomain.domain }),
                        href: "/domains/" + mydomain.domain,
                        color: "success",
                        timeout: 5000,
                    });
                }

                if (!allImportInProgress) refreshDomains();
            },
            (error) => {
                domain.wait = false;
                throw error;
            },
        );
    }

    let allImportInProgress = false;
    async function importAllDomains() {
        if (importableDomainsList) {
            allImportInProgress = true;
            for (const d of importableDomainsList.filter((dn) => dn.indexOf($filteredName) >= 0)) {
                if (!haveDomain($domains_idx, d)) {
                    await importDomain({ domain: d, wait: false }, true);
                }
            }
            allImportInProgress = false;
            refreshDomains();
        }
    }

    let createDomainInProgress = false;
    async function createDomainOnProvider() {
        createDomainInProgress = true;
        try {
            await createDomain(provider, fqdn($filteredName, ""))
            await importDomain({ domain: fqdn($filteredName, ""), wait: false })
            createDomainInProgress = false;
        } catch (err) {
            createDomainInProgress = false;
            throw err;
        }
    }
</script>

<Card {...$$restProps}>
    {#if !noDomainsList && !discoveryError}
        <CardHeader>
            <div class="d-flex justify-content-between align-items-center">
                <div>
                    {@html $t("provider.provider", {
                        provider:
                            "<em>" +
                            escape(
                                provider._comment
                                    ? provider._comment
                                    : $providersSpecs
                                      ? $providersSpecs[provider._srctype].name
                                      : "",
                            ) +
                            "</em>",
                    })}
                </div>
                {#if importableDomainsList != null}
                    <Button
                        type="button"
                        color="secondary"
                        disabled={allImportInProgress}
                        size="sm"
                        on:click={importAllDomains}
                    >
                        {#if allImportInProgress}
                            <Spinner size="sm" />
                        {/if}
                        {$t("provider.import-domains")}
                    </Button>
                {/if}
            </div>
        </CardHeader>
    {/if}
    {#if importableDomainsList == null}
        <div class="d-flex justify-content-center align-items-center gap-2 my-3">
            <Spinner color="primary" />
            {$t("wait.asking-domains")}
        </div>
    {:else}
        <ListGroup flush>
            {#if importableDomainsList.length == 0}
                {#if discoveryError}
                    <ListGroupItem class="mx-2 my-3">
                        <p class="text-danger">
                            <Icon
                                name="exclamation-octagon-fill"
                                class="float-start display-5 me-2"
                            />
                            {discoveryError}
                        </p>
                        <div class="text-center">
                            <Button href={"/providers/" + encodeURIComponent(provider._id)} outline>
                                {$t("provider.check-config")}
                            </Button>
                        </div>
                    </ListGroupItem>
                {:else if noDomainsList}
                    <ListGroupItem class="text-center my-3">
                        {$t("errors.domain-list")}
                    </ListGroupItem>
                {:else if !importableDomainsList || importableDomainsList.length === 0}
                    <ListGroupItem class="text-center my-3">
                        {$t("errors.domain-have")}
                    </ListGroupItem>
                {:else if importableDomainsList.length === 0}
                    <ListGroupItem class="text-center my-3">
                        {#if $providersSpecs}
                            {$t("errors.domain-all-imported", {
                                provider: $providersSpecs[provider._srctype].name,
                            })}
                        {/if}
                    </ListGroupItem>
                {/if}
            {:else}
                {#each importableDomainsList.map((dn) => ({
                    domain: dn,
                    id_provider: provider._id,
                    wait: false,
                })).filter((dn) => dn.domain.indexOf($filteredName) >= 0) as domain}
                    <ListGroupItem class="d-flex justify-content-between align-items-center text-muted">
                        <DomainWithProvider {domain} />
                        <div>
                            {#if haveDomain($domains_idx, domain.domain)}
                                <Badge class="ml-1" color="success">
                                    <Icon name="check" />
                                    {$t("onboarding.import.imported")}
                                </Badge>
                            {:else}
                                <Button
                                    type="button"
                                    class="ml-1"
                                    color="primary"
                                    size="sm"
                                    disabled={domain.wait || allImportInProgress}
                                          on:click={() => importDomain(domain, false)}
                                >
                                    {#if domain.wait}
                                        <Spinner size="sm" />
                                    {/if}
                                    {$t("domains.add-now")}
                                </Button>
                            {/if}
                        </div>
                    </ListGroupItem>
                {/each}
                {#if importableDomainsList.filter((dn) => dn.indexOf($filteredName) >= 0).length != importableDomainsList.length}
                    <ListGroupItem
                        tag="button"
                        class="text-center text-muted"
                        on:click={() => $filteredName = ""}
                    >
                        {$t('domains.and-more-filtered', { count: importableDomainsList.length - importableDomainsList.filter((dn) => dn.indexOf($filteredName) >= 0).length })}
                    </ListGroupItem>
                {/if}
            {/if}
            {#if $filteredName && $providersSpecs && $providersSpecs[provider._srctype] && $providersSpecs[provider._srctype].capabilities.indexOf('CreateDomain') >= 0 && !importableDomainsList.filter((dn) => dn == $filteredName).length}
                <ListGroupItem class="d-flex justify-content-between align-items-center">
                    <DomainWithProvider class="text-muted fst-italic" domain={{domain: fqdn($filteredName, ""), id_provider: provider._id, wait: false}} />
                    <div>
                        <Button
                            type="button"
                            class="ml-1"
                            color="warning"
                            size="sm"
                            disabled={createDomainInProgress || !validateDomain($filteredName)}
                            on:click={createDomainOnProvider}
                        >
                            {#if createDomainInProgress}
                                <Spinner size="sm" />
                            {/if}
                            {$t("domains.create-on-provider", {provider: provider._comment ? provider._comment : $providersSpecs ? $providersSpecs[provider._srctype].name : ""})}
                        </Button>
                    </div>
                </ListGroupItem>
            {/if}
        </ListGroup>
    {/if}
</Card>
