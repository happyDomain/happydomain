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
    import {
        Alert,
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        Col,
        Container,
        Form,
        FormGroup,
        Icon,
        Row,
    } from "@sveltestrap/sveltestrap";
    import { page } from "$app/state";

    import { toasts } from "$lib/stores/toasts";
    import { getChecksByCnameOptions, putChecksByCnameOptions } from "$lib/api-admin";
    import { getCheckStatus } from "$lib/api/checks";
    import Resource from "$lib/components/inputs/Resource.svelte";
    import CheckerOptionsGroups from "$lib/components/checkers/CheckerOptionsGroups.svelte";

    let cname = $derived(page.params.cname!);

    let checkerStatusQ = $derived(getCheckStatus(cname));
    let checkerOptionsQ = $derived(getChecksByCnameOptions({ path: { cname } }));
    let optionValues = $state<Record<string, any>>({});
    let saving = $state(false);

    $effect(() => {
        checkerOptionsQ.then((optionsR) => {
            optionValues = { ...((optionsR.data as Record<string, unknown>) || {}) };
        });
    });

    async function saveOptions() {
        saving = true;
        try {
            await putChecksByCnameOptions({
                path: { cname },
                body: { options: optionValues },
            });
            checkerOptionsQ = getChecksByCnameOptions({ path: { cname } });
            toasts.addToast({
                message: `Plugin options updated successfully`,
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: "Failed to update options: " + error,
                timeout: 10000,
            });
        } finally {
            saving = false;
        }
    }

    async function cleanOrphanedOptions(adminOpts: any[]) {
        const validOptIds = new Set(adminOpts.map((opt) => opt.id));
        const cleanedOptions: Record<string, any> = {};

        for (const [key, value] of Object.entries(optionValues)) {
            if (validOptIds.has(key)) {
                cleanedOptions[key] = value;
            }
        }

        saving = true;
        try {
            await putChecksByCnameOptions({
                path: { cname },
                body: { options: cleanedOptions },
            });
            checkerOptionsQ = getChecksByCnameOptions({ path: { cname } });
            toasts.addToast({
                message: `Orphaned options removed successfully`,
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: "Failed to clean options: " + error,
                timeout: 10000,
            });
        } finally {
            saving = false;
        }
    }

    function getOrphanedOptions(adminOpts: any[]): string[] {
        const validOptIds = new Set(adminOpts.map((opt) => opt.id));
        return Object.keys(optionValues).filter((key) => !validOptIds.has(key));
    }
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <Button color="link" href="/checks" class="mb-2">
                <Icon name="arrow-left"></Icon>
                Back to checkers
            </Button>
            <h1 class="display-5">
                <Icon name="puzzle-fill"></Icon>
                {cname}
            </h1>
        </Col>
    </Row>

    {#await checkerStatusQ}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                Loading checker status...
            </p>
        </Card>
    {:then status}
        {#if status}
            <Row class="mb-4">
                <Col md={6}>
                    <Card>
                        <CardHeader>
                            <strong>Checker Information</strong>
                        </CardHeader>
                        <CardBody>
                            <dl class="row mb-0">
                                <dt class="col-sm-4">Name:</dt>
                                <dd class="col-sm-8">{status.name}</dd>

                                <dt class="col-sm-4">Availability:</dt>
                                <dd class="col-sm-8">
                                    {#if status.availableOn}
                                        <div class="d-flex flex-wrap gap-1">
                                            {#if status.availableOn.applyToDomain}
                                                <Badge color="success">Domain-level</Badge>
                                            {/if}
                                            {#if status.availableOn.limitToProviders && status.availableOn.limitToProviders.length > 0}
                                                <Badge color="primary">
                                                    Providers: {status.availableOn.limitToProviders.join(
                                                        ", ",
                                                    )}
                                                </Badge>
                                            {/if}
                                            {#if status.availableOn.limitToServices && status.availableOn.limitToServices.length > 0}
                                                <Badge color="info">
                                                    Services: {status.availableOn.limitToServices.join(
                                                        ", ",
                                                    )}
                                                </Badge>
                                            {/if}
                                            {#if !status.availableOn.applyToDomain && (!status.availableOn.limitToProviders || status.availableOn.limitToProviders.length === 0) && (!status.availableOn.limitToServices || status.availableOn.limitToServices.length === 0)}
                                                <Badge color="secondary">General</Badge>
                                            {/if}
                                        </div>
                                    {:else}
                                        <Badge color="secondary">General</Badge>
                                    {/if}
                                </dd>
                            </dl>
                        </CardBody>
                    </Card>
                </Col>

                <Col md={6}>
                    {#await checkerOptionsQ}
                        <Card>
                            <CardBody>
                                <p class="text-center mb-0">
                                    <span class="spinner-border spinner-border-sm me-2"></span>
                                    Loading options...
                                </p>
                            </CardBody>
                        </Card>
                    {:then _optionsR}
                        {@const adminOpts = status.options?.adminOpts || []}
                        {@const readOnlyOptGroups = [
                            {
                                key: "userOpts",
                                label: "User Options",
                                opts: status.options?.userOpts || [],
                            },
                            {
                                key: "domainOpts",
                                label: "Domain Options",
                                opts: status.options?.domainOpts || [],
                            },
                            {
                                key: "serviceOpts",
                                label: "Service Options",
                                opts: status.options?.serviceOpts || [],
                            },
                            {
                                key: "runOpts",
                                label: "Run Options",
                                opts: status.options?.runOpts || [],
                            },
                        ]}
                        {@const hasAnyOpts =
                            adminOpts.length > 0 ||
                            readOnlyOptGroups.some((g) => g.opts.length > 0)}
                        {@const orphanedOpts = getOrphanedOptions(adminOpts)}

                        {#if orphanedOpts.length > 0}
                            <Alert color="warning" class="mb-3">
                                <div class="d-flex justify-content-between align-items-center">
                                    <div>
                                        <Icon name="exclamation-triangle-fill"></Icon>
                                        <strong>Orphaned options detected:</strong>
                                        {orphanedOpts.join(", ")}
                                    </div>
                                    <Button
                                        color="danger"
                                        size="sm"
                                        onclick={() => cleanOrphanedOptions(adminOpts)}
                                        disabled={saving}
                                    >
                                        <Icon name="trash"></Icon>
                                        Clean Up
                                    </Button>
                                </div>
                            </Alert>
                        {/if}

                        {#if adminOpts.length > 0}
                            <Card class="mb-3">
                                <CardHeader>
                                    <strong>Admin Options</strong>
                                </CardHeader>
                                <CardBody>
                                    <Form on:submit={saveOptions}>
                                        {#each adminOpts as optDoc}
                                            {#if optDoc.id}
                                                {@const optName = optDoc.id}
                                                <FormGroup>
                                                    <Resource
                                                        edit={true}
                                                        index={optName}
                                                        specs={optDoc}
                                                        type={optDoc.type || "string"}
                                                        bind:value={optionValues[optName]}
                                                    />
                                                </FormGroup>
                                            {/if}
                                        {/each}
                                        <div class="d-flex gap-2">
                                            <Button type="submit" color="success" disabled={saving}>
                                                {#if saving}
                                                    <span
                                                        class="spinner-border spinner-border-sm me-1"
                                                    ></span>
                                                {/if}
                                                <Icon name="check-circle"></Icon>
                                                Save Changes
                                            </Button>
                                        </div>
                                    </Form>
                                </CardBody>
                            </Card>
                        {/if}

                        <CheckerOptionsGroups groups={readOnlyOptGroups} />

                        {#if !hasAnyOpts}
                            <Card>
                                <CardBody>
                                    <Alert color="info" class="mb-0">
                                        <Icon name="info-circle"></Icon>
                                        This checker has no configurable options.
                                    </Alert>
                                </CardBody>
                            </Card>
                        {/if}
                    {:catch error}
                        <Card>
                            <CardBody>
                                <Alert color="danger" class="mb-0">
                                    <Icon name="exclamation-triangle-fill"></Icon>
                                    Error loading options: {error.message}
                                </Alert>
                            </CardBody>
                        </Card>
                    {/await}
                </Col>
            </Row>
        {:else}
            <Alert color="danger">
                <Icon name="exclamation-triangle-fill"></Icon>
                Error: checker data not found
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill"></Icon>
            Error loading checker: {error.message}
        </Alert>
    {/await}
</Container>
