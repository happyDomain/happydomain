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
    import { onDestroy, untrack } from "svelte";

    import {
        Button,
        Icon,
        Input,
        InputGroup,
        InputGroupText,
        Spinner,
    } from "@sveltestrap/sveltestrap";
    import { t } from "$lib/translations";
    import { fetchSSHHostKeys } from "$lib/api/resolver";
    import type { HappydnsSshHostKey } from "$lib/api-base/types.gen";
    import type { Domain } from "$lib/model/domain";
    import type { SvcsSSHFPsBody } from "$lib/services_bodies";

    type SSHFPEntry = NonNullable<SvcsSSHFPsBody["SSHFP"]>[number];

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: SvcsSSHFPsBody;
    }

    let { dn, origin, readonly = false, value = $bindable() }: Props = $props();

    // RFC 4255 sec. 3.1 and its successors. The numbers are what travels in the
    // record; the names are what an administrator recognizes.
    const ALGORITHMS = [
        { value: 1, label: "RSA" },
        { value: 2, label: "DSA" },
        { value: 3, label: "ECDSA" },
        { value: 4, label: "Ed25519" },
        { value: 6, label: "Ed448" },
    ];
    const FINGERPRINT_TYPES = [
        { value: 1, label: "SHA-1" },
        { value: 2, label: "SHA-256" },
    ];

    // Read back through the body on each access: the state proxy hands out the
    // reactive array, which a plain assignment expression would not.
    if (!value.SSHFP) value.SSHFP = [];
    const fingerprints = (): SSHFPEntry[] => value.SSHFP ?? [];

    function addSSHFP() {
        fingerprints().push({ algorithm: 4, type: 2, fingerprint: "" });
    }

    function deleteSSHFP(idx: number) {
        fingerprints().splice(idx, 1);
    }

    // The host these fingerprints describe: the name the service is attached
    // to, which is the one a client will look up. Read once, as the seed of the
    // field below, which the user is then free to edit.
    const defaultHost = untrack(() =>
        [dn, origin.domain].filter(Boolean).join(".").replace(/\.+$/, ""),
    );

    let host = $state(defaultHost);
    let port = $state(22);
    let retrieving = $state(false);
    let retrieveError: string | null = $state(null);
    let retrieveInfo: string | null = $state(null);
    let retrieveAbort: AbortController | null = null;

    onDestroy(() => retrieveAbort?.abort());

    function alreadyPublished(algorithm: number, type: number, fingerprint: string): boolean {
        return fingerprints().some(
            (entry) =>
                entry.algorithm === algorithm &&
                entry.type === type &&
                (entry.fingerprint || "").toLowerCase() === fingerprint,
        );
    }

    /**
     * Publishes both fingerprints of each key: the SHA-256 one is what
     * validating clients use, the SHA-1 one is still asked for by a few
     * deployments and costs nothing to carry.
     */
    function importKeys(keys: HappydnsSshHostKey[]): number {
        let added = 0;

        const addFingerprint = (algorithm: number, type: number, fingerprint: string | undefined) => {
            const normalized = (fingerprint || "").toLowerCase();
            if (!normalized || alreadyPublished(algorithm, type, normalized)) return;

            fingerprints().push({ algorithm, type, fingerprint: normalized });
            added++;
        };

        for (const key of keys) {
            const algorithm = key.algorithm ?? 0;
            addFingerprint(algorithm, 2, key.sha256);
            addFingerprint(algorithm, 1, key.sha1);
        }

        return added;
    }

    // The statuses the backend reports, other than "ok" (see
    // internal/usecase/ssh_hostkeys.go classifySSHError).
    const KNOWN_STATUSES = new Set(["blocked", "connect-error", "handshake-error", "timeout"]);

    /** Turns a backend status into a sentence, falling back to its error message. */
    function statusMessage(status: string | undefined, errorMsg: string | undefined): string {
        if (status && KNOWN_STATUSES.has(status)) {
            return $t(`sshfp.retrieve.status.${status}`);
        }
        return errorMsg || status || "";
    }

    async function retrieve() {
        retrieveAbort?.abort();
        const controller = new AbortController();
        retrieveAbort = controller;

        retrieving = true;
        retrieveError = null;
        retrieveInfo = null;

        try {
            const resp = await fetchSSHHostKeys({ host, port }, controller.signal);
            if (controller.signal.aborted) return;

            if (resp.status !== "ok") {
                retrieveError = statusMessage(resp.status, resp.errorMsg);
                return;
            }

            const added = importKeys(resp.keys ?? []);
            retrieveInfo =
                added > 0
                    ? $t("sshfp.retrieve.added", { count: added })
                    : $t("sshfp.retrieve.nothing-new");
        } catch (err) {
            if (controller.signal.aborted) return;
            retrieveError = (err as Error).message;
        } finally {
            if (!controller.signal.aborted) retrieving = false;
            if (retrieveAbort === controller) retrieveAbort = null;
        }
    }
</script>

{#if !readonly}
    <div class="mb-3">
        <p class="text-muted small mb-2">{$t("sshfp.retrieve.description")}</p>
        <InputGroup>
            <InputGroupText>{$t("sshfp.retrieve.host")}</InputGroupText>
            <Input bind:value={host} placeholder={defaultHost} disabled={retrieving} />
            <InputGroupText>{$t("sshfp.retrieve.port")}</InputGroupText>
            <Input
                type="number"
                style="max-width: 8em"
                bind:value={port}
                min="1"
                max="65535"
                disabled={retrieving}
            />
            <Button
                type="button"
                color="primary"
                outline
                onclick={retrieve}
                disabled={retrieving || !host}
            >
                {#if retrieving}
                    <Spinner size="sm" type="border" />
                {:else}
                    <Icon name="download" />
                {/if}
                {$t("sshfp.retrieve.action")}
            </Button>
        </InputGroup>
        {#if retrieveError}
            <div class="alert alert-danger py-2 px-3 mt-2 mb-0">
                <Icon name="exclamation-triangle-fill" />
                {retrieveError}
            </div>
        {:else if retrieveInfo}
            <div class="alert alert-success py-2 px-3 mt-2 mb-0">
                <Icon name="check-circle-fill" />
                {retrieveInfo}
            </div>
        {/if}
    </div>
{/if}

<table class="table table-striped table-hover">
    <thead>
        <tr>
            <th>{$t("sshfp.algorithm")}</th>
            <th>{$t("sshfp.type")}</th>
            <th>{$t("sshfp.fingerprint")}</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        {#if fingerprints().length}
            {#each fingerprints() as fingerprint, idx (fingerprint)}
                <tr>
                    <td>
                        <Input type="select" bsSize="sm" bind:value={fingerprint.algorithm}>
                            {#each ALGORITHMS as algorithm (algorithm.value)}
                                <option value={algorithm.value}>{algorithm.label}</option>
                            {/each}
                        </Input>
                    </td>
                    <td>
                        <Input type="select" bsSize="sm" bind:value={fingerprint.type}>
                            {#each FINGERPRINT_TYPES as type (type.value)}
                                <option value={type.value}>{type.label}</option>
                            {/each}
                        </Input>
                    </td>
                    <td>
                        <Input bsSize="sm" bind:value={fingerprint.fingerprint} />
                    </td>
                    <td>
                        <Button
                            type="button"
                            color="danger"
                            outline
                            size="sm"
                            onclick={() => deleteSSHFP(idx)}
                        >
                            <Icon name="trash" />
                        </Button>
                    </td>
                </tr>
            {/each}
        {/if}
    </tbody>
    <tfoot>
        <tr>
            <td colspan="4">
                <Button type="button" color="primary" outline size="sm" onclick={addSSHFP}>
                    <Icon name="plus" />
                    {$t("common.new-row")}
                </Button>
            </td>
        </tr>
    </tfoot>
</table>
