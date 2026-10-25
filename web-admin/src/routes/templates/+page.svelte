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
    import { Button, Col, Container, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    import ModalNewTemplate, {
        controls as ctrlNewTemplate,
    } from "$lib/components/zone_templates/ModalNewTemplate.svelte";
    import TemplateTable from "$lib/components/zone_templates/TemplateTable.svelte";
    import { createGlobalTemplate, listGlobalTemplates } from "./backend";

    let templatesQ = $state(listGlobalTemplates());
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="files"></Icon>
                Zone Templates
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead">Templates offered to every user</span>
                {#await templatesQ then templates}
                    <span>Total: {templates.length} templates</span>
                {/await}
            </p>
        </Col>
        <Col md={4} class="d-flex flex-column gap-2 align-items-end justify-content-center">
            <Button color="primary" on:click={() => ctrlNewTemplate.Open()}>
                <Icon name="plus-lg"></Icon>
                New template
            </Button>
            <Button color="secondary" outline on:click={() => ctrlNewTemplate.OpenImport()}>
                <Icon name="upload"></Icon>
                Import a zone file
            </Button>
        </Col>
    </Row>

    {#await templatesQ}
        <div class="text-center py-5"><Spinner /></div>
    {:then templates}
        {#if templates.length === 0}
            <p class="text-center text-muted">No global template yet: users only see their own.</p>
        {:else}
            <!-- Global templates have no edit nor delete button in the list: they are managed from their page. -->
            <TemplateTable items={templates} onedit={() => {}} ondelete={() => {}} />
        {/if}
    {:catch err}
        <div class="alert alert-danger">
            Unable to load the templates: {err}
            <Button
                size="sm"
                color="danger"
                outline
                on:click={() => (templatesQ = listGlobalTemplates())}
            >
                Try again
            </Button>
        </div>
    {/await}
</Container>

<ModalNewTemplate create={createGlobalTemplate} />
