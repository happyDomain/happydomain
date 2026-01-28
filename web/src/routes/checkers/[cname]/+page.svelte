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
        Form,
        FormGroup,
        Icon,
        Row,
    } from "@sveltestrap/sveltestrap";
    import { page } from "$app/state";

    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { getCheckStatus, getCheckOptions, updateCheckOptions } from "$lib/api/checks";
    import Resource from "$lib/components/inputs/Resource.svelte";
    import CheckerOptionsGroups from "$lib/components/checkers/CheckerOptionsGroups.svelte";

    let cname = $derived(page.params.cname!);

    let checkStatusPromise = $derived(getCheckStatus(cname));
    let checkOptionsPromise = $derived(getCheckOptions(cname));
    let optionValues = $state<Record<string, any>>({});
    let resolvedStatus = $state<any>(null);
    let saving = $state(false);

    $effect(() => {
        checkStatusPromise.then((status) => {
            resolvedStatus = status;
        });
    });

    $effect(() => {
        checkOptionsPromise.then((options) => {
            optionValues = { ...(options || {}) };
        });
    });

    async function saveOptions() {
        saving = true;
        try {
            await updateCheckOptions(cname, optionValues);
            checkOptionsPromise = getCheckOptions(cname);
            toasts.addToast({
                message: $t("checkers.messages.options-updated"),
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.messages.update-failed", { error: String(error) }),
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
            await updateCheckOptions(cname, cleanedOptions);
            checkOptionsPromise = getCheckOptions(cname);
            toasts.addToast({
                message: $t("checkers.messages.options-cleaned"),
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.messages.clean-failed", { error: String(error) }),
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
    <title>{resolvedStatus?.name ?? cname} - {$t("checkers.title")} - happyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle title={resolvedStatus?.name ?? cname}></PageTitle>

    {#await checkStatusPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.loading-info")}
            </p>
        </Card>
    {:then status}
        {#if status}
            <Row class="mb-4">
                <Col md={6}>
                    <Card>
                        <CardHeader>
                            <strong>{$t("checkers.detail.checker-information")}</strong>
                        </CardHeader>
                        <CardBody>
                            <dl class="row mb-0">
                                <dt class="col-sm-4">{$t("checkers.detail.name")}</dt>
                                <dd class="col-sm-8">{status.name}</dd>

                                <dt class="col-sm-4">{$t("checkers.detail.availability")}</dt>
                                <dd class="col-sm-8">
                                    {#if status.availability}
                                        <div class="d-flex flex-wrap gap-1">
                                            {#if status.availability.applyToDomain}
                                                <Badge color="success">
                                                    {$t("checkers.availability.domain-level")}
                                                </Badge>
                                            {/if}
                                            {#if status.availability.limitToProviders && status.availability.limitToProviders.length > 0}
                                                <Badge color="primary">
                                                    {$t("checkers.availability.providers", {
                                                        providers:
                                                            status.availability.limitToProviders.join(
                                                                ", ",
                                                            ),
                                                    })}
                                                </Badge>
                                            {/if}
                                            {#if status.availability.limitToServices && status.availability.limitToServices.length > 0}
                                                <Badge color="info">
                                                    {$t("checkers.availability.services", {
                                                        services:
                                                            status.availability.limitToServices.join(
                                                                ", ",
                                                            ),
                                                    })}
                                                </Badge>
                                            {/if}
                                            {#if !status.availability.applyToDomain && (!status.availability.limitToProviders || status.availability.limitToProviders.length === 0) && (!status.availability.limitToServices || status.availability.limitToServices.length === 0)}
                                                <Badge color="secondary">
                                                    {$t("checkers.availability.general")}
                                                </Badge>
                                            {/if}
                                        </div>
                                    {:else}
                                        <Badge color="secondary">
                                            {$t("checkers.availability.general")}
                                        </Badge>
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
                                    {$t("checkers.detail.loading-options")}
                                </p>
                            </CardBody>
                        </Card>
                    {:then options}
                        {@const userOpts = status.options?.userOpts || []}
                        {@const readOnlyOptGroups = [
                            {
                                key: "adminOpts",
                                label: $t("checkers.option-groups.global-settings"),
                                opts: status.options?.adminOpts || [],
                            },
                            {
                                key: "domainOpts",
                                label: $t("checkers.option-groups.domain-settings"),
                                opts: status.options?.domainOpts || [],
                            },
                            {
                                key: "serviceOpts",
                                label: $t("checkers.option-groups.service-settings"),
                                opts: status.options?.serviceOpts || [],
                            },
                            {
                                key: "runOpts",
                                label: $t("checkers.option-groups.checker-parameters"),
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
                                        {$t("checkers.detail.orphaned-options", {
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
                                        {$t("checkers.detail.clean-up")}
                                    </Button>
                                </div>
                            </Alert>
                        {/if}

                        {#if userOpts.length > 0}
                            <Card class="mb-3">
                                <CardHeader>
                                    <strong>{$t("checkers.detail.configuration")}</strong>
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
                                                {$t("checkers.detail.save-changes")}
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
                                        {$t("checkers.detail.no-configurable-options")}
                                    </Alert>
                                </CardBody>
                            </Card>
                        {/if}
                    {:catch error}
                        <Card>
                            <CardBody>
                                <Alert color="danger" class="mb-0">
                                    <Icon name="exclamation-triangle-fill"></Icon>
                                    {$t("checkers.detail.error-loading-options", {
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
                {$t("checkers.checker-info-not-found")}
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill"></Icon>
            {$t("checkers.error-loading-checker", { error: error.message })}
        </Alert>
    {/await}
</div>
