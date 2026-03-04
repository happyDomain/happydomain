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
    import { onMount } from "svelte";

    import { Button, Col, Container, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import ProviderList from "$lib/components/providers/List.svelte";
    import { appConfig, navigate } from "$lib/stores/config";
    import { domains, refreshDomains } from "$lib/stores/domains";
    import { providers, refreshProviders } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    if (!$domains) {
        refreshDomains();
    }
    refreshProviders();

    onMount(() => {
        if (page.url.searchParams.has("newProvider")) {
            newProviderControls.Open();
        }
    });
</script>

<svelte:head>
    <title>{$t("provider.title")} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-5">
    <Row>
        <Col md={{ size: 8, offset: 2 }}>
            <PageTitle title={$t("provider.title")} subtitle={$t("provider.description")}>
                {#if !$appConfig.disable_providers}
                    <div class="d-flex justify-content-end mb-2">
                        <Button
                            type="button"
                            color="primary"
                            on:click={() => selectProviderTypeControls.Open()}
                        >
                            <Icon name="plus" />
                            {$t("common.add-new-thing", { thing: $t("provider.kind") })}
                        </Button>
                    </div>
                {/if}
            </PageTitle>

            {#if !$providers}
                <div class="d-flex justify-content-center mt-5">
                    <Spinner color="primary" />
                </div>
            {:else}
                <div class="mt-5">
                    <ProviderList
                        items={$providers}
                        on:new-provider={() => selectProviderTypeControls.Open()}
                        on:click={(event) =>
                            navigate("providers/" + encodeURIComponent(event.detail._id))}
                    />
                </div>
            {/if}
        </Col>
    </Row>
</Container>
