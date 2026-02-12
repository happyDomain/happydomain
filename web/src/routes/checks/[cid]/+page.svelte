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

    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";
    import { getCheckStatus, getCheckOptions, updateCheckOptions } from "$lib/api/checks";
    import Resource from "$lib/components/inputs/Resource.svelte";
    import CheckerOptionsGroups from "$lib/components/checkers/CheckerOptionsGroups.svelte";

    let cid = $derived(page.params.cid!);

    let checkStatusPromise = $derived(getCheckStatus(cid));
    let checkOptionsPromise = $derived(getCheckOptions(cid));
    let optionValues = $state<Record<string, any>>({});
    let saving = $state(false);

    $effect(() => {
        checkOptionsPromise.then((options) => {
            optionValues = { ...(options || {}) };
        });
    });

    async function saveOptions() {
        saving = true;
        try {
            await updateCheckOptions(cid, optionValues);
            checkOptionsPromise = getCheckOptions(cid);
            toasts.addToast({
                message: $t("checks.messages.options-updated"),
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checks.messages.update-failed", { error: String(error) }),
                timeout: 10000,
            });
        } finally {
            saving = false;
        }
    }

    async function cleanOrphanedOptions(userOpts: any[]) {
        const validOptIds = new Set(userOpts.map((opt) => opt.id));
        const cleanedOptions: Record<string, any> = {};

        for (const [key, value] of Object.entries(optionValues)) {
            if (validOptIds.has(key)) {
                cleanedOptions[key] = value;
            }
        }

        saving = true;
        try {
            await updateCheckOptions(cid, cleanedOptions);
            checkOptionsPromise = getCheckOptions(cid);
            toasts.addToast({
                message: $t("checks.messages.options-cleaned"),
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checks.messages.clean-failed", { error: String(error) }),
                timeout: 10000,
            });
        } finally {
            saving = false;
        }
    }

    function getOrphanedOptions(userOpts: any[], readOnlyOptGroups: any[]): string[] {
        const validOptIds = new Set(userOpts.map((opt) => opt.id));

        for (const group of readOnlyOptGroups) {
            for (const opt of group.opts) {
                validOptIds.add(opt.id);
            }
        }

        return Object.keys(optionValues).filter((key) => !validOptIds.has(key));
    }
</script>

<svelte:head>
    <title>{cid} - {$t("checks.title")} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <Button color="link" href="/checks" class="mb-2">
                <Icon name="arrow-left"></Icon>
                {$t("checks.back-to-checks")}
            </Button>
            <h1 class="display-5">
                <Icon name="check-circle-fill"></Icon>
                {cid}
            </h1>
        </Col>
    </Row>

    {#await checkStatusPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checks.loading-info")}
            </p>
        </Card>
    {:then status}
        {#if status}
            <Row class="mb-4">
                <Col md={6}>
                    <Card>
                        <CardHeader>
                            <strong>{$t("checks.detail.check-information")}</strong>
                        </CardHeader>
                        <CardBody>
                            <dl class="row mb-0">
                                <dt class="col-sm-4">{$t("checks.detail.name")}</dt>
                                <dd class="col-sm-8">{status.name}</dd>

                                <dt class="col-sm-4">{$t("checks.detail.availability")}</dt>
                                <dd class="col-sm-8">
                                    {#if status.availability}
                                        <div class="d-flex flex-wrap gap-1">
                                            {#if status.availability.applyToDomain}
                                                <Badge color="success"
                                                    >{$t("checks.availability.domain-level")}</Badge
                                                >
                                            {/if}
                                            {#if status.availability.limitToProviders && status.availability.limitToProviders.length > 0}
                                                <Badge color="primary">
                                                    {$t("checks.availability.providers", {
                                                        providers:
                                                            status.availability.limitToProviders.join(
                                                                ", ",
                                                            ),
                                                    })}
                                                </Badge>
                                            {/if}
                                            {#if status.availability.limitToServices && status.availability.limitToServices.length > 0}
                                                <Badge color="info">
                                                    {$t("checks.availability.services", {
                                                        services:
                                                            status.availability.limitToServices.join(
                                                                ", ",
                                                            ),
                                                    })}
                                                </Badge>
                                            {/if}
                                            {#if !status.availability.applyToDomain && (!status.availability.limitToProviders || status.availability.limitToProviders.length === 0) && (!status.availability.limitToServices || status.availability.limitToServices.length === 0)}
                                                <Badge color="secondary"
                                                    >{$t("checks.availability.general")}</Badge
                                                >
                                            {/if}
                                        </div>
                                    {:else}
                                        <Badge color="secondary"
                                            >{$t("checks.availability.general")}</Badge
                                        >
                                    {/if}
                                </dd>
                            </dl>
                        </CardBody>
                    </Card>
                </Col>

                <Col md={6}>
                    {#await checkOptionsPromise}
                        <Card>
                            <CardBody>
                                <p class="text-center mb-0">
                                    <span class="spinner-border spinner-border-sm me-2"></span>
                                    {$t("checks.detail.loading-options")}
                                </p>
                            </CardBody>
                        </Card>
                    {:then options}
                        {@const userOpts = status.options?.userOpts || []}
                        {@const readOnlyOptGroups = [
                            {
                                key: "adminOpts",
                                label: $t("checks.option-groups.global-settings"),
                                opts: status.options?.adminOpts || [],
                            },
                            {
                                key: "domainOpts",
                                label: $t("checks.option-groups.domain-settings"),
                                opts: status.options?.domainOpts || [],
                            },
                            {
                                key: "serviceOpts",
                                label: $t("checks.option-groups.service-settings"),
                                opts: status.options?.serviceOpts || [],
                            },
                            {
                                key: "runOpts",
                                label: $t("checks.option-groups.check-parameters"),
                                opts: status.options?.runOpts || [],
                            },
                        ]}
                        {@const hasAnyOpts =
                            userOpts.length > 0 || readOnlyOptGroups.some((g) => g.opts.length > 0)}
                        {@const orphanedOpts = getOrphanedOptions(userOpts, readOnlyOptGroups)}

                        {#if orphanedOpts.length > 0}
                            <Alert color="warning" class="mb-3">
                                <div class="d-flex justify-content-between align-items-center">
                                    <div>
                                        <Icon name="exclamation-triangle-fill"></Icon>
                                        {$t("checks.detail.orphaned-options", {
                                            options: orphanedOpts.join(", "),
                                        })}
                                    </div>
                                    <Button
                                        color="danger"
                                        size="sm"
                                        onclick={() => cleanOrphanedOptions(userOpts)}
                                        disabled={saving}
                                    >
                                        <Icon name="trash"></Icon>
                                        {$t("checks.detail.clean-up")}
                                    </Button>
                                </div>
                            </Alert>
                        {/if}

                        {#if userOpts.length > 0}
                            <Card class="mb-3">
                                <CardHeader>
                                    <strong>{$t("checks.detail.configuration")}</strong>
                                </CardHeader>
                                <CardBody>
                                    <Form
                                        on:submit={(e) => {
                                            e.preventDefault();
                                            saveOptions();
                                        }}
                                    >
                                        {#each userOpts as optDoc}
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
                                                {$t("checks.detail.save-changes")}
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
                                        {$t("checks.detail.no-configurable-options")}
                                    </Alert>
                                </CardBody>
                            </Card>
                        {/if}
                    {:catch error}
                        <Card>
                            <CardBody>
                                <Alert color="danger" class="mb-0">
                                    <Icon name="exclamation-triangle-fill"></Icon>
                                    {$t("checks.detail.error-loading-options", {
                                        error: error.message,
                                    })}
                                </Alert>
                            </CardBody>
                        </Card>
                    {/await}
                </Col>
            </Row>
        {:else}
            <Alert color="danger">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("checks.check-info-not-found")}
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill"></Icon>
            {$t("checks.error-loading-check", { error: error.message })}
        </Alert>
    {/await}
</Container>
