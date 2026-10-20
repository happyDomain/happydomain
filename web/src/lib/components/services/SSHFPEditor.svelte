<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    import type { dnsTypeSSHFP } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import {
        KNOWN_STATUSES,
        SHA256,
        defaultHost as seedHost,
        importKeys,
        newSSHFP,
    } from "$lib/services/sshfp";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        // Shared by svcs.SSHFPs and abstract.Server, which both keep their
        // records under SSHFP.
        value: { SSHFP?: dnsTypeSSHFP[] };
    }

    let { dn, origin, readonly = false, value = $bindable() }: Props = $props();

    // IANA "SSHFP RR Parameters" (RFC 4255, 6594, 7479, 8709). Number 5 is
    // unassigned. DSA stays selectable for existing records although Retrieve
    // can never produce it (see sshfpAlgorithms in internal/usecase/ssh_hostkeys.go).
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
    const fingerprints = (): dnsTypeSSHFP[] => value.SSHFP ?? [];

    function addSSHFP() {
        fingerprints().push(newSSHFP(4, SHA256, ""));
    }

    function deleteSSHFP(idx: number) {
        fingerprints().splice(idx, 1);
    }

    // Read once, as the seed of the field below, which the user is then free
    // to edit.
    const defaultHost = untrack(() => seedHost(dn, origin.domain));

    let host = $state(defaultHost);
    let port = $state(22);
    let retrieving = $state(false);
    let retrieveError: string | null = $state(null);
    let retrieveInfo: string | null = $state(null);
    let retrieveAbort: AbortController | null = null;

    onDestroy(() => retrieveAbort?.abort());

    // The statuses the backend reports, other than "ok", that have a sentence.
    const KNOWN = new Set(KNOWN_STATUSES);

    /** Turns a backend status into a sentence, falling back to its error message. */
    function statusMessage(status: string | undefined, errorMsg: string | undefined): string {
        if (status && KNOWN.has(status)) {
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

            const added = importKeys(fingerprints(), resp.keys ?? []);
            retrieveInfo =
                added > 0
                    ? $t("sshfp.retrieve.added", { count: added })
                    : $t("sshfp.retrieve.nothing-new");
        } catch (err) {
            if (controller.signal.aborted) return;
            retrieveError = (err as Error).message;
        } finally {
            // An aborted run was superseded by a newer one, which owns the spinner.
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
            <!-- Keyed by row object so a delete or an import does not rebind the inputs to the wrong row. -->
            {#each fingerprints() as fingerprint, idx (fingerprint)}
                <tr>
                    <td>
                        <Input type="select" bsSize="sm" bind:value={fingerprint.Algorithm}>
                            {#each ALGORITHMS as algorithm (algorithm.value)}
                                <option value={algorithm.value}>{algorithm.label}</option>
                            {/each}
                        </Input>
                    </td>
                    <td>
                        <Input type="select" bsSize="sm" bind:value={fingerprint.Type}>
                            {#each FINGERPRINT_TYPES as type (type.value)}
                                <option value={type.value}>{type.label}</option>
                            {/each}
                        </Input>
                    </td>
                    <td>
                        <Input bsSize="sm" bind:value={fingerprint.FingerPrint} />
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
