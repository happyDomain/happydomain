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
        Icon,
        ListGroup,
        ListGroupItem,
        Row,
    } from "@sveltestrap/sveltestrap";
    import { page } from "$app/state";

    import { toasts } from "$lib/stores/toasts";
    import {
        getCheckersByCheckerId,
        getCheckersByCheckerIdOptions,
        putCheckersByCheckerIdOptions,
    } from "$lib/api-base";
    import type { HappydnsCheckerOptionDocumentation } from "$lib/api-base";
    import ResourceInput from "$lib/components/inputs/Resource.svelte";
    import { availabilityBadges, formatDuration, getOrphanedOptionKeys, filterValidOptions } from "$lib/utils";

    let checkerId = $derived(page.params.checkerId!);

    let checkerQ = $derived(getCheckersByCheckerId({ path: { checkerId } }));
    let checkerOptionsQ = $derived(getCheckersByCheckerIdOptions({ path: { checkerId } }));
    let optionValues = $state<Record<string, unknown>>({});
    let saving = $state(false);

    $effect(() => {
        checkerOptionsQ.then((optionsR) => {
            optionValues = { ...((optionsR.data as Record<string, unknown>) || {}) };
        });
    });

    async function saveOptions() {
        saving = true;
        try {
            await putCheckersByCheckerIdOptions({
                path: { checkerId },
                body: optionValues,
            });
            checkerOptionsQ = getCheckersByCheckerIdOptions({ path: { checkerId } });
            toasts.addToast({
                message: `Checker options updated successfully`,
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

    async function cleanOrphanedOptions(adminOpts: HappydnsCheckerOptionDocumentation[]) {
        saving = true;
        try {
            await putCheckersByCheckerIdOptions({
                path: { checkerId },
                body: filterValidOptions(optionValues, adminOpts),
            });
            checkerOptionsQ = getCheckersByCheckerIdOptions({ path: { checkerId } });
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

</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <Button color="link" href="/checkers" class="mb-2">
                <Icon name="arrow-left"></Icon>
                Back to checkers
            </Button>
            <h1 class="display-5">
                <Icon name="puzzle-fill"></Icon>
                {checkerId}
            </h1>
        </Col>
    </Row>

    {#await checkerQ}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                Loading checker status...
            </p>
        </Card>
    {:then checkerR}
        {@const checker = checkerR.data}
        {#if checker}
            <Row class="mb-4">
                <Col md={6}>
                    <Card class="mb-3">
                        <CardHeader>
                            <strong>Checker Information</strong>
                        </CardHeader>
                        <CardBody>
                            <dl class="row mb-0">
                                <dt class="col-sm-4">Name:</dt>
                                <dd class="col-sm-8">{checker.name}</dd>

                                <dt class="col-sm-4">Availability:</dt>
                                <dd class="col-sm-8">
                                    {#if availabilityBadges(checker.availability).length > 0}
                                        <div class="d-flex flex-wrap gap-1">
                                            {#each availabilityBadges(checker.availability) as badge}
                                                <Badge color={badge.color}
                                                    >{badge.label}-level</Badge
                                                >
                                            {/each}
                                        </div>
                                    {:else}
                                        <Badge color="secondary">General</Badge>
                                    {/if}
                                    {#if checker.availability?.limitToProviders?.length}
                                        <div class="mt-1 small text-muted">
                                            Providers: {checker.availability.limitToProviders.join(
                                                ", ",
                                            )}
                                        </div>
                                    {/if}
                                    {#if checker.availability?.limitToServices?.length}
                                        <div class="mt-1 small text-muted">
                                            Services: {checker.availability.limitToServices.join(
                                                ", ",
                                            )}
                                        </div>
                                    {/if}
                                </dd>

                                {#if checker.interval}
                                    <dt class="col-sm-4">Interval:</dt>
                                    <dd class="col-sm-8">
                                        <span>default {formatDuration(checker.interval.default)}</span>
                                        <span class="text-muted small ms-2">
                                            (min {formatDuration(checker.interval.min)} / max {formatDuration(checker.interval.max)})
                                        </span>
                                    </dd>
                                {/if}
                            </dl>
                        </CardBody>
                    </Card>

                    {#if checker.rules && checker.rules.length > 0}
                        <Card>
                            <CardHeader class="d-flex align-items-center justify-content-between">
                                <div>
                                    <strong>Check Rules</strong>
                                    <Badge color="secondary" class="ms-2">
                                        {checker.rules.length}
                                    </Badge>
                                </div>
                                {#if checker.rules.reduce((acc, rule) => acc + rule.options?.adminOpts?.length, 0) > 0}
                                    <Button
                                        color="success"
                                        size="sm"
                                        onclick={saveOptions}
                                        disabled={saving}
                                    >
                                        {#if saving}
                                            <span class="spinner-border spinner-border-sm me-1"
                                            ></span>
                                        {:else}
                                            <Icon name="check-circle"></Icon>
                                        {/if}
                                        Save
                                    </Button>
                                {/if}
                            </CardHeader>
                            <ListGroup flush>
                                {#each checker.rules as rule, i}
                                    {@const ruleOpts = rule.options?.adminOpts || []}
                                    <ListGroupItem>
                                        <div class="d-flex align-items-start gap-2 mb-1">
                                            <Icon
                                                name="check2-circle"
                                                class="text-success mt-1 flex-shrink-0"
                                            ></Icon>
                                            <div class="flex-grow-1">
                                                <strong>{rule.name}</strong>
                                                {#if rule.description}
                                                    <p class="text-muted small mb-0">
                                                        {rule.description}
                                                    </p>
                                                {/if}
                                            </div>
                                        </div>
                                        {#if ruleOpts.length > 0}
                                            <div class="ms-4 mt-2">
                                                <Form onsubmit={saveOptions}>
                                                    {#each ruleOpts as optDoc, index}
                                                        {#if optDoc.id}
                                                            <ResourceInput
                                                                edit
                                                                index={"" + index}
                                                                specs={optDoc}
                                                                type={optDoc.type || "string"}
                                                                bind:value={optionValues[optDoc.id]}
                                                            />
                                                        {/if}
                                                    {/each}
                                                </Form>
                                            </div>
                                        {/if}
                                    </ListGroupItem>
                                {/each}
                            </ListGroup>
                        </Card>
                    {/if}
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
                        {@const adminOpts = checker.options?.adminOpts || []}
                        {@const readOnlyOptGroups = [
                            {
                                key: "userOpts",
                                label: "User Options",
                                opts: checker.options?.userOpts || [],
                            },
                            {
                                key: "domainOpts",
                                label: "Domain Options",
                                opts: checker.options?.domainOpts || [],
                            },
                            {
                                key: "serviceOpts",
                                label: "Service Options",
                                opts: checker.options?.serviceOpts || [],
                            },
                            {
                                key: "runOpts",
                                label: "Run Options",
                                opts: checker.options?.runOpts || [],
                            },
                        ]}
                        {@const rulesAdminOpts = (checker.rules || []).flatMap(
                            (r) => r.options?.adminOpts || [],
                        )}
                        {@const allAdminOpts = [...adminOpts, ...rulesAdminOpts]}
                        {@const hasAnyOpts =
                            allAdminOpts.length > 0 ||
                            readOnlyOptGroups.some((g) => g.opts.length > 0)}
                        {@const orphanedOpts = getOrphanedOptionKeys(optionValues, allAdminOpts)}

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
                                        onclick={() => cleanOrphanedOptions(allAdminOpts)}
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
                                <CardHeader
                                    class="d-flex align-items-center justify-content-between"
                                >
                                    <strong>Admin Options</strong>
                                    <Button
                                        form="adminoptsform"
                                        color="success"
                                        size="sm"
                                        onclick={saveOptions}
                                        disabled={saving}
                                    >
                                        {#if saving}
                                            <span class="spinner-border spinner-border-sm me-1"
                                            ></span>
                                        {:else}
                                            <Icon name="check-circle"></Icon>
                                        {/if}
                                        Save
                                    </Button>
                                </CardHeader>
                                <CardBody>
                                    <Form id="adminoptsform" onsubmit={saveOptions}>
                                        {#each adminOpts as optDoc, index}
                                            {#if optDoc.id}
                                                <ResourceInput
                                                    edit
                                                    index={"" + index}
                                                    specs={optDoc}
                                                    type={optDoc.type || "string"}
                                                    bind:value={optionValues[optDoc.id]}
                                                />
                                            {/if}
                                        {/each}
                                    </Form>
                                </CardBody>
                            </Card>
                        {/if}

                        {#each readOnlyOptGroups.filter((g) => g.opts.length > 0) as group}
                            <Card class="mb-3">
                                <CardHeader>
                                    <strong>{group.label}</strong>
                                    <Badge color="secondary" class="ms-2">read-only</Badge>
                                </CardHeader>
                                <CardBody>
                                    <dl class="row mb-0">
                                        {#each group.opts as opt}
                                            <dt class="col-sm-4">{opt.label || opt.id}</dt>
                                            <dd class="col-sm-8">
                                                <span class="text-muted small"
                                                    >{opt.type || "string"}</span
                                                >
                                                {#if opt.description}
                                                    <div class="form-text">{opt.description}</div>
                                                {/if}
                                            </dd>
                                        {/each}
                                    </dl>
                                </CardBody>
                            </Card>
                        {/each}

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
