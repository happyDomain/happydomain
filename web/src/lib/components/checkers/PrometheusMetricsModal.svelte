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
        Button,
        Icon,
        Input,
        InputGroup,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
    } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        isOpen?: boolean;
        url: string;
    }

    let { isOpen = $bindable(false), url }: Props = $props();

    let copied = $state(false);
    let resetTimer: ReturnType<typeof setTimeout> | undefined;

    let parsedUrl = $derived.by(() => {
        try {
            return new URL(url, window.location.origin);
        } catch {
            return null;
        }
    });
    let exampleHost = $derived(parsedUrl?.host ?? window.location.host);
    let examplePath = $derived(parsedUrl?.pathname ?? url);

    function toggle(): void {
        isOpen = !isOpen;
    }

    async function copyUrl(): Promise<void> {
        try {
            await navigator.clipboard.writeText(url);
            copied = true;
            if (resetTimer) clearTimeout(resetTimer);
            resetTimer = setTimeout(() => (copied = false), 2000);
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.list.prometheus-metrics-copy-failed", {
                    error: String(error),
                }),
                timeout: 5000,
            });
        }
    }
</script>

<Modal {isOpen} {toggle} size="lg">
    <ModalHeader {toggle}>
        {$t("checkers.list.prometheus-metrics-modal.title")}
    </ModalHeader>
    <ModalBody>
        <InputGroup>
            <Input type="text" value={url} readonly />
            <Button color="primary" onclick={copyUrl}>
                <Icon name={copied ? "clipboard-check" : "clipboard"}></Icon>
                {copied
                    ? $t("checkers.list.prometheus-metrics-modal.copied")
                    : $t("checkers.list.prometheus-metrics-modal.copy")}
            </Button>
        </InputGroup>

        <p class="text-muted small mt-3 mb-2">
            {$t("checkers.list.prometheus-metrics-modal.description")}
        </p>

        <pre class="bg-body-secondary p-2 rounded small mb-0"><code>{$t(
                "checkers.list.prometheus-metrics-modal.example",
                { host: exampleHost, path: examplePath },
            )}</code></pre>
    </ModalBody>
    <ModalFooter>
        <Button color="secondary" onclick={toggle}>
            {$t("checkers.list.prometheus-metrics-modal.close")}
        </Button>
    </ModalFooter>
</Modal>
