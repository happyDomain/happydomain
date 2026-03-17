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
        Open() { },
    };
</script>

<script lang="ts">
    import { Modal, ModalBody, ModalHeader, Spinner } from "@sveltestrap/sveltestrap";

    import { getDomainInfo } from "$lib/api/domaininfo";
    import type { DomainInfo } from "$lib/model/domaininfo";
    import { t } from "$lib/translations";
    import DomainInfoDisplay from "$lib/components/DomainInfoDisplay.svelte";

    interface Props {
        domain: string;
        isOpen?: boolean;
    }

    let { domain, isOpen = $bindable(false) }: Props = $props();

    let info: DomainInfo | null = $state(null);
    let error_response: string | null = $state(null);
    let request_pending = $state(false);

    function Open(): void {
        isOpen = true;
        info = null;
        error_response = null;
        request_pending = true;

        getDomainInfo(domain).then(
            (result) => {
                info = result;
                request_pending = false;
            },
            (error: unknown) => {
                const msg = error instanceof Error ? error.message : String(error);
                error_response = msg;
                request_pending = false;
            },
        );
    }

    function toggle(): void {
        isOpen = !isOpen;
    }

    controls.Open = Open;
</script>

<Modal {isOpen} size="lg" {toggle}>
    <ModalHeader {toggle}>
        {$t("domaininfo.page-title")} — <span class="font-monospace">{domain}</span>
    </ModalHeader>
    <ModalBody>
        {#if request_pending}
            <div class="text-center text-muted py-4">
                <Spinner />
                <p class="mt-3">{$t("common.spinning")}…</p>
            </div>
        {:else if error_response !== null}
            <div class="card border-danger">
                <div class="card-body">
                    <div class="d-flex align-items-center">
                        <i class="bi bi-x-circle text-danger fs-3 me-3"></i>
                        <p class="card-text mb-0">{error_response}</p>
                    </div>
                </div>
            </div>
        {:else if info !== null}
            <DomainInfoDisplay {info} {domain} />
        {/if}
    </ModalBody>
</Modal>
