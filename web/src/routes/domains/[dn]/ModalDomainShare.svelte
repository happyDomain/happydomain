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

<script module lang="ts">
    import type { ModalController } from "$lib/model/modal_controller";

    export const controls: ModalController = {
        Open() {},
    };
</script>

<script lang="ts">
    import {
        Alert,
        Button,
        Icon,
        Input,
        ListGroup,
        ListGroupItem,
        Modal,
        ModalBody,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { getDomainShareStatus, shareDomain, unshareDomain } from "$lib/api/domains";
    import type {
        HappydnsDomainShareUser,
        HappydnsProviderShareUser,
    } from "$lib/api-base/types.gen";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        isOpen?: boolean;
    }

    let { domain, isOpen = $bindable(false) }: Props = $props();

    let shares: Array<HappydnsDomainShareUser> = $state([]);
    let providerShares: Array<HappydnsProviderShareUser> = $state([]);
    let loading = $state(false);
    let inviting = $state(false);
    let error: string | null = $state(null);
    let userRef = $state("");
    let withProvider = $state(false);
    // Set when the owner explicitly asks to change a provider access the
    // invitee already has: until then the checkbox reflects the current state.
    let overrideProvider = $state(false);

    function matches(ref: string, id?: string, email?: string): boolean {
        return ref === id || (!!email && ref.toLowerCase() === email.toLowerCase());
    }

    // The invitee, when they are already known: either shared on this domain,
    // or holding a grant on its provider through another domain.
    let matchedShare = $derived(shares.find((s) => matches(userRef.trim(), s.id, s.email)));
    let matchedProviderShare = $derived(
        providerShares.find((s) => matches(userRef.trim(), s.id, s.email)),
    );

    // The provider access this invitee currently has, or null when they are
    // unknown and the owner is free to choose. Provider grants are keyed by
    // (provider, grantee) with no domain component, so this is a single
    // provider-wide state, not a per-domain one.
    let lockedState: boolean | null = $derived(
        matchedShare ? (matchedShare.with_provider ?? false) : matchedProviderShare ? true : null,
    );

    // The other domains that a provider-wide revoke would also affect.
    let affectedDomains = $derived(
        (matchedProviderShare?.domains ?? []).filter((d) => d.id !== domain.id),
    );

    $effect(() => {
        if (lockedState !== null && !overrideProvider) {
            withProvider = lockedState;
        }
    });

    // A different invitee means a different current state: never carry over the
    // previous override.
    let overriddenFor = "";
    $effect(() => {
        const ref = userRef.trim();
        if (ref !== overriddenFor) {
            overriddenFor = ref;
            overrideProvider = false;
        }
    });

    async function fetchShares(): Promise<void> {
        loading = true;
        try {
            const status = await getDomainShareStatus(domain.id);
            shares = status.shares ?? [];
            providerShares = status.provider_shares ?? [];
        } catch (err: any) {
            error = err?.message || String(err);
        } finally {
            loading = false;
        }
    }

    async function reload(): Promise<void> {
        error = null;
        await fetchShares();
    }

    function Open(): void {
        isOpen = true;
        userRef = "";
        withProvider = false;
        overrideProvider = false;
        error = null;
        shares = [];
        providerShares = [];
        reload();
    }

    function toggle(): void {
        isOpen = !isOpen;
    }

    async function invite(): Promise<void> {
        if (!userRef.trim()) return;
        inviting = true;
        error = null;
        try {
            await shareDomain(domain.id, userRef.trim(), withProvider);
            userRef = "";
            withProvider = false;
            overrideProvider = false;
            await reload();
        } catch (err: any) {
            error = err?.message || String(err);
            await fetchShares();
        } finally {
            inviting = false;
        }
    }

    async function revoke(userId: string): Promise<void> {
        error = null;
        try {
            await unshareDomain(domain.id, userId);
            await reload();
        } catch (err: any) {
            error = err?.message || String(err);
        }
    }

    controls.Open = Open;
</script>

<Modal {isOpen} size="lg" {toggle}>
    <ModalHeader {toggle}>
        {$t("domains.share.title")} <span class="font-monospace">{domain.domain}</span>
    </ModalHeader>
    <ModalBody>
        <p class="text-muted">{$t("domains.share.description")}</p>

        <form
            onsubmit={(e) => {
                e.preventDefault();
                invite();
            }}
        >
            <label class="form-label" for="share-user">{$t("domains.share.user-label")}</label>
            <div class="d-flex gap-2">
                <Input
                    id="share-user"
                    type="text"
                    bind:value={userRef}
                    placeholder={$t("domains.share.user-placeholder")}
                    disabled={inviting}
                />
                <Button type="submit" color="primary" disabled={inviting || !userRef.trim()}>
                    {#if inviting}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="person-plus" />
                    {/if}
                    {$t("domains.share.invite")}
                </Button>
            </div>
            <div class="form-check mt-2">
                <input
                    class="form-check-input"
                    type="checkbox"
                    id="share-with-provider"
                    bind:checked={withProvider}
                    disabled={inviting || (lockedState !== null && !overrideProvider)}
                />
                <label class="form-check-label" for="share-with-provider">
                    {$t("domains.share.with-provider")}
                </label>
            </div>
            {#if lockedState !== null}
                <div class="d-flex align-items-center gap-2 mt-1">
                    <small class="form-text text-muted">
                        {#if overrideProvider}
                            {$t("domains.share.provider-overridden")}
                        {:else if lockedState}
                            {$t("domains.share.provider-locked-on")}
                        {:else}
                            {$t("domains.share.provider-locked-off")}
                        {/if}
                    </small>
                    {#if overrideProvider}
                        <Button
                            outline
                            size="sm"
                            color="secondary"
                            type="button"
                            title={$t("domains.share.provider-keep")}
                            on:click={() => (overrideProvider = false)}
                        >
                            <Icon name="arrow-counterclockwise" />
                        </Button>
                    {:else}
                        <Button
                            outline
                            size="sm"
                            color="secondary"
                            type="button"
                            title={$t("domains.share.provider-override")}
                            on:click={() => (overrideProvider = true)}
                        >
                            <Icon name="pencil" />
                        </Button>
                    {/if}
                </div>
            {/if}
            {#if overrideProvider && !withProvider && affectedDomains.length > 0}
                <Alert color="warning" class="py-2 small mb-0 mt-2">
                    {$t("domains.share.provider-revoke-warning", {
                        domains: affectedDomains.map((d) => d.domain).join(", "),
                    })}
                </Alert>
            {/if}
        </form>

        {#if error !== null}
            <div class="alert alert-danger mt-3" role="alert">{error}</div>
        {/if}

        <hr />

        <h6>{$t("domains.share.current")}</h6>
        {#if loading}
            <div class="text-center text-muted py-3">
                <Spinner size="sm" />
            </div>
        {:else if shares.length === 0}
            <p class="text-muted">{$t("domains.share.none")}</p>
        {:else}
            <ListGroup>
                {#each shares as share (share.id)}
                    <ListGroupItem class="d-flex justify-content-between align-items-center">
                        <span class="d-flex align-items-center gap-2">
                            {share.email}
                            {#if share.with_provider}
                                <Icon
                                    name="key-fill"
                                    class="text-success"
                                    title={$t("domains.share.provider-shared")}
                                />
                            {:else}
                                <Icon
                                    name="key"
                                    class="text-muted"
                                    title={$t("domains.share.provider-not-shared")}
                                />
                            {/if}
                        </span>
                        <Button
                            outline
                            size="sm"
                            color="danger"
                            on:click={() => share.id && revoke(share.id)}
                        >
                            <Icon name="person-dash" />
                            {$t("domains.share.revoke")}
                        </Button>
                    </ListGroupItem>
                {/each}
            </ListGroup>
        {/if}
    </ModalBody>
</Modal>
