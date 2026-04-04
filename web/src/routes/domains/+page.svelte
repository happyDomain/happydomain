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
    import { page } from "$app/state";
    import {
        Button,
        Col,
        Container,
        Icon,
        Input,
        InputGroup,
        InputGroupText,
        Row,
    } from "@sveltestrap/sveltestrap";

    import { addDomain } from "$lib/api/domains";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import DomainTable from "$lib/components/domains/Table.svelte";
    import NewDomainModal, {
        controls as newDomainControls,
    } from "$lib/components/modals/NewDomain.svelte";
    import PickProvider, {
        controls as pickProviderControls,
    } from "$lib/components/modals/PickProvider.svelte";
    import type { Provider } from "$lib/model/provider";
    import { navigate } from "$lib/stores/config";
    import { domains, refreshDomains } from "$lib/stores/domains";
    import { providers } from "$lib/stores/providers";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    const newDomainName = page.url.searchParams.get("new");

    let searchQuery = $state("");
    let selectedProviderId = $state(page.url.searchParams.get("provider") ?? "");
    let selectedGroup = $state<string | null>(
        page.url.searchParams.has("group") ? page.url.searchParams.get("group") : null,
    );

    const availableGroups = $derived(
        [...new Set(($domains ?? []).map((d) => d.group ?? ""))].sort(),
    );

    $effect(() => {
        const params = new URLSearchParams(window.location.search);
        if (selectedProviderId) {
            params.set("provider", selectedProviderId);
        } else {
            params.delete("provider");
        }
        if (selectedGroup !== null) {
            params.set("group", selectedGroup);
        } else {
            params.delete("group");
        }
        const newSearch = params.toString();
        const currentSearch = window.location.search.replace(/^\?/, "");
        if (newSearch !== currentSearch) {
            const newUrl = window.location.pathname + (newSearch ? `?${newSearch}` : "");
            navigate(newUrl, { replaceState: true, keepFocus: true, noScroll: true });
        }
    });

    $effect(() => {
        if (newDomainName) {
            pickProviderControls.Open();
        }
    });

    async function onNewDomainProviderSelected(provider: Provider) {
        const domain = await addDomain(newDomainName!, provider);
        toasts.addToast({
            title: $t("domains.attached-new"),
            message: $t("domains.added-success", { domain: domain.domain }),
            href: "/domains/" + domain.domain,
            type: "success",
            timeout: 5000,
        });
        refreshDomains();
        navigate("/domains/" + domain.domain);
    }
</script>

<svelte:head>
    <title>{$t("domains.title")} - happyDomain</title>
</svelte:head>

<NewDomainModal />
<PickProvider ondone={onNewDomainProviderSelected} />

<Container class="flex-fill my-5">
    <PageTitle title={$t("domains.title")} subtitle={$t("domains.description")}>
        <div class="d-flex justify-content-end mb-2">
            <Button type="button" color="primary" onclick={() => newDomainControls.Open()}>
                <Icon name="plus" />
                {$t("common.add-new-thing", { thing: $t("domains.kind") })}
            </Button>
        </div>
    </PageTitle>

    <Row class="mb-4 mt-3">
        <Col md={8} lg={6}>
            <InputGroup>
                <InputGroupText>
                    <Icon name="search"></Icon>
                </InputGroupText>
                <Input
                    type="text"
                    placeholder={$t("domains.search-placeholder")}
                    bind:value={searchQuery}
                />
            </InputGroup>
        </Col>
        <Col md={4} lg={3}>
            <Input type="select" bind:value={selectedProviderId}>
                <option value="">{$t("provider.all")}</option>
                {#each $providers ?? [] as provider (provider._id)}
                    <option value={provider._id}>{provider._comment || provider._id}</option>
                {/each}
            </Input>
        </Col>
        <Col md={4} lg={3}>
            <Input
                type="select"
                value={selectedGroup ?? ""}
                onchange={(e) => {
                    const v = (e.target as HTMLSelectElement).value;
                    selectedGroup = v === "\x00" ? null : v;
                }}
            >
                <option value={"\x00"}>{$t("domaingroups.all")}</option>
                {#each availableGroups as group (group)}
                    <option value={group}
                        >{group === "" ? $t("domaingroups.no-group") : group}</option
                    >
                {/each}
            </Input>
        </Col>
    </Row>

    <div class="mt-5">
        <DomainTable
            items={($domains ?? []).filter(
                (d) =>
                    d.domain.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1 &&
                    (!selectedProviderId || d.id_provider === selectedProviderId) &&
                    (selectedGroup === null ||
                        d.group === selectedGroup ||
                        ((selectedGroup === "" || selectedGroup === "undefined") &&
                            (d.group === "" || d.group === undefined))),
            )}
        />
    </div>
</Container>
