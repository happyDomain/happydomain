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
    import { onMount } from "svelte";
    import { Button, Card, Col, Container, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    import {
        getSecretsStatus,
        postSecretsDropSafes,
        postSecretsReseal,
        postSecretsRewrap,
    } from "$lib/api-admin";
    import type { SecretResealReport, UsecaseSecretsStatus } from "$lib/api-admin";
    import { toasts } from "$lib/stores/toasts";

    import InstanceKeysTable from "./InstanceKeysTable.svelte";
    import OwnerKeystoresCard from "./OwnerKeystoresCard.svelte";
    import PolicyCard from "./PolicyCard.svelte";
    import ResealReportTable from "./ResealReportTable.svelte";
    import SecretActionCard from "./SecretActionCard.svelte";
    import SecretCountsTable from "./SecretCountsTable.svelte";
    import { sealedCount } from "./secrets";

    let status = $state<UsecaseSecretsStatus | null>(null);
    let loading = $state(true);
    let error = $state<string | null>(null);

    async function fetchStatus() {
        loading = status === null;
        error = null;
        try {
            const { data, error: err } = await getSecretsStatus();
            if (err) throw new Error((err as { errmsg?: string })?.errmsg || String(err));
            status = data ?? null;
        } catch (e) {
            error = e instanceof Error ? e.message : "Unknown error";
        } finally {
            loading = false;
        }
    }

    onMount(fetchStatus);

    // The keys of the instance keyset, among those of every store.
    const keys = $derived((status?.keys ?? []).filter((k) => k.kek === "instance"));
    // MISSING and UNREADABLE rows count safes no key opens, they are not keys.
    const hasKeyset = $derived(
        keys.some((k) => k.status !== "MISSING" && k.status !== "UNREADABLE"),
    );

    const totals = $derived(
        Object.values(status?.objects ?? {}).reduce(
            (acc, c) => ({
                clear: acc.clear + (c.clear ?? 0),
                sealed: acc.sealed + sealedCount(c),
                // Only these need the keyset: the others stay in the
                // keystores their owners chose.
                sealedInstance: acc.sealedInstance + (c.sealed?.instance ?? 0),
                unreadable: acc.unreadable + (c.unreadable ?? 0),
                undecodable: acc.undecodable + (c.undecodable ?? 0),
            }),
            { clear: 0, sealed: 0, sealedInstance: 0, unreadable: 0, undecodable: 0 },
        ),
    );

    // Some credentials are not stored the way the policy stores new ones.
    const resealNeeded = $derived(
        (status?.policy === "instance" && totals.clear > 0) ||
            (status?.policy === "plaintext" && totals.sealedInstance > 0),
    );

    // Mirrors the checks of DropSafes on the server, which refuses anyway.
    const dropSafesBlocker = $derived.by(() => {
        if (status?.policy !== "plaintext")
            return "Switch to the plaintext policy, restart, and reseal first.";
        if (totals.sealedInstance + totals.unreadable > 0)
            return "Credentials are still sealed under the keyset: reseal first, or have the unreadable ones entered again.";
        if (totals.undecodable > 0)
            return "Some objects could not be looked at and may hold sealed credentials: repair or delete them first.";
        return undefined;
    });

    function droppedSafes(n: number) {
        toasts.addToast({
            type: "info",
            message: `${n} user key${n === 1 ? "" : "s"} deleted. The keyset can now be removed from the configuration.`,
            timeout: 10000,
        });
        fetchStatus();
    }
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <div class="d-flex justify-content-between align-items-start">
                <div>
                    <h1 class="display-5">
                        <Icon name="shield-lock"></Icon>
                        Secrets
                    </h1>
                    <p class="text-muted lead">
                        Follow how credentials are stored, and migrate them
                    </p>
                </div>
                <Button size="sm" color="secondary" outline onclick={fetchStatus}>
                    <Icon name="arrow-clockwise"></Icon> Refresh
                </Button>
            </div>
        </Col>
    </Row>

    {#if error}
        <Card color="danger" body class="mb-4">
            <Icon name="exclamation-triangle-fill"></Icon>
            {error}
        </Card>
    {/if}

    {#if loading}
        <div class="d-flex align-items-center gap-2">
            <Spinner size="sm" />
            <span>Loading secrets status...</span>
        </div>
    {:else if status}
        <Row class="mb-4 g-4">
            <Col lg={5}>
                <PolicyCard class="h-100" policy={status.policy ?? ""} {hasKeyset} />
            </Col>
            <Col lg={7}>
                <InstanceKeysTable class="h-100" {keys} />
            </Col>
        </Row>

        <SecretCountsTable class="mb-4" objects={status.objects ?? {}} />

        <OwnerKeystoresCard class="mb-4" />

        <h2 class="h4 mb-3">Migrations</h2>
        <p class="text-muted">
            Run them with a single HappyDomain process on the database. They can be interrupted:
            running them again resumes the work.
        </p>
        <Row class="g-4">
            <Col md={4}>
                <SecretActionCard
                    title="Reseal"
                    icon="arrow-repeat"
                    color={resealNeeded ? "primary" : "secondary"}
                    label="Reseal credentials"
                    confirmText="Store every credential the way the current policy stores new ones? Make sure a single HappyDomain process is running."
                    run={postSecretsReseal}
                    onDone={fetchStatus}
                >
                    Stores every credential the way the current policy stores new ones: encrypted
                    under
                    <code>instance</code>, in clear under <code>plaintext</code>.
                    {#if resealNeeded}
                        <strong class="d-block mt-2 text-primary">
                            Some credentials are not stored as the policy requires.
                        </strong>
                    {/if}
                    {#snippet result(reports: SecretResealReport[])}
                        <ResealReportTable {reports} />
                    {/snippet}
                </SecretActionCard>
            </Col>
            <Col md={4}>
                <SecretActionCard
                    title="Rewrap"
                    icon="key"
                    label="Rewrap users' keys"
                    confirmText="Re-encrypt every user's key under the primary instance key?"
                    run={postSecretsRewrap}
                    disabled={!hasKeyset}
                    disabledReason="No instance keyset configured."
                    onDone={fetchStatus}
                >
                    After <code>happydomain secret-keyset rotate</code> and a restart, re-encrypts
                    the users' keys under the new primary key. The credentials themselves are not
                    rewritten.
                    {#snippet result(report: SecretResealReport)}
                        <ResealReportTable reports={[report]} />
                    {/snippet}
                </SecretActionCard>
            </Col>
            <Col md={4}>
                <SecretActionCard
                    title="Drop safes"
                    icon="trash"
                    color="danger"
                    label="Drop users' keys"
                    confirmText="Delete every user's key and the keyset check record? Backups taken before still need the keyset."
                    run={postSecretsDropSafes}
                    disabled={dropSafesBlocker !== undefined}
                    disabledReason={dropSafesBlocker}
                    onDone={droppedSafes}
                >
                    Once back to clear, deletes the users' keys and the keyset check record: the
                    keyset is then no longer needed at startup.
                </SecretActionCard>
            </Col>
        </Row>
    {/if}
</Container>
