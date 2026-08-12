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
        Dropdown,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        Input,
        Label,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
    } from "@sveltestrap/sveltestrap";

    import { buildDiagnosticsReport } from "$lib/stores/diagnostics";
    import { FORGES, issueURL, mailtoURL, reportMarkdown } from "$lib/utils/report";
    import { t } from "$lib/translations";

    let isOpen = $state(false);
    let description = $state("");
    let diagnostics = $state("");
    let showDiagnostics = $state(false);
    let copied = $state(false);
    let copyFailed = $state(false);

    const toggle = () => (isOpen = !isOpen);

    /**
     * Open the report dialog. When it is opened from a failure the user just
     * saw, that error travels with it: they shouldn't have to describe an
     * error message we already have in hand.
     */
    export async function open(reportedError?: string) {
        description = "";
        diagnostics = $t("report.collecting");
        copied = false;
        copyFailed = false;
        showDiagnostics = false;
        isOpen = true;

        diagnostics = await buildDiagnosticsReport(reportedError);
    }

    async function copyReport() {
        try {
            if (!navigator.clipboard) throw new Error("clipboard unavailable");
            await navigator.clipboard.writeText(reportMarkdown(description, diagnostics));
            copied = true;
            copyFailed = false;
        } catch {
            // Without a secure context, or if the browser denies the
            // permission, there is no clipboard: reveal the report so the
            // user can select it by hand, and say so explicitly.
            showDiagnostics = true;
            copied = false;
            copyFailed = true;
        }
    }
</script>

<Modal {isOpen} {toggle} size="lg" scrollable>
    <ModalHeader {toggle}>
        <Icon name="bug" class="me-2" />
        {$t("report.title")}
    </ModalHeader>
    <ModalBody>
        <p class="text-muted">
            {$t("report.intro")}
        </p>

        <Label for="report-description" class="fw-bold">{$t("report.what-happened")}</Label>
        <Input
            id="report-description"
            type="textarea"
            rows={4}
            autofocus
            bind:value={description}
            placeholder={$t("report.placeholder")}
        />

        <div class="mt-3">
            <Button
                color="link"
                class="p-0 text-decoration-none"
                on:click={() => (showDiagnostics = !showDiagnostics)}
            >
                <Icon name={showDiagnostics ? "chevron-down" : "chevron-right"} />
                {$t("report.attached-details")}
            </Button>
            <small class="text-muted d-block">{$t("report.attached-details-desc")}</small>
            {#if showDiagnostics}
                <Input
                    class="mt-2 font-monospace small"
                    type="textarea"
                    rows={10}
                    bind:value={diagnostics}
                />
            {/if}
        </div>
    </ModalBody>
    <ModalFooter class="justify-content-between">
        <div>
            <Button color="secondary" outline on:click={copyReport}>
                <Icon name={copied ? "clipboard-check" : "clipboard"} class="me-1" />
                {copied ? $t("report.copied") : $t("report.copy")}
            </Button>
            {#if copyFailed}
                <small class="text-danger d-block mt-1">{$t("report.copy-failed")}</small>
            {/if}
        </div>
        <div class="d-flex gap-2">
            <Button
                color="primary"
                href={issueURL(FORGES[0], description, diagnostics)}
                target="_blank"
                rel="noopener"
                on:click={toggle}
            >
                <Icon name="box-arrow-up-right" class="me-1" />
                {$t("report.open-issue", { forge: FORGES[0].name })}
            </Button>
            <Dropdown>
                <DropdownToggle color="primary" outline caret>
                    {$t("report.other-forge")}
                </DropdownToggle>
                <DropdownMenu end>
                    {#each FORGES.slice(1) as forge (forge.name)}
                        <DropdownItem
                            href={issueURL(forge, description, diagnostics)}
                            target="_blank"
                            rel="noopener"
                            on:click={toggle}
                        >
                            {forge.name}
                        </DropdownItem>
                    {/each}
                    <DropdownItem divider />
                    <DropdownItem href={mailtoURL(description, diagnostics)} on:click={toggle}>
                        <Icon name="envelope" class="me-2" />
                        {$t("report.by-email")}
                    </DropdownItem>
                </DropdownMenu>
            </Dropdown>
        </div>
    </ModalFooter>
</Modal>
