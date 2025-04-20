<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
    import { createEventDispatcher } from "svelte";

    import { Button, Icon, Input, Label, ModalFooter, Spinner } from "@sveltestrap/sveltestrap";

    import { getServiceRecords } from "$lib/api/zone";
    import HelpButton from "$lib/components/Help.svelte";
    import TableRecords from "$lib/components/domains/TableRecords.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import { locale, t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    export let toggle: () => void;
    export let step: number;
    export let service: ServiceCombined | null = null;
    export let form = "addSvcForm";
    export let origin: Domain | undefined = undefined;
    export let update = false;
    export let zoneId: number | undefined = undefined;
    export let canDelete = false;
    export let canContinue = false;

    export let addServiceInProgress = false;
    export let deleteServiceInProgress = false;

    let helpHref = "";
    $: {
        if (service && service._svctype) {
            const svcPart = service._svctype.toLowerCase().split(".");
            if (svcPart.length === 2) {
                if (svcPart[0] === "svcs") {
                    helpHref = "records/" + svcPart[1].toUpperCase() + "/";
                } else if (svcPart[0] === "abstract") {
                    helpHref = "services/" + svcPart[1] + "/";
                } else if (svcPart[0] === "provider") {
                    helpHref = "services/providers/" + svcPart[1] + "/";
                } else {
                    helpHref = svcPart[svcPart.length - 1] + "/";
                }
            } else {
                helpHref = svcPart[svcPart.length - 1] + "/";
            }
        } else {
            helpHref = "";
        }
        helpHref = "https://help.happydomain.org/" + $locale + "/" + helpHref;
    }

    let showRecords = false;

    let recordsHeight = 120;
    let recordsHeightResize = false;
    function resizeRecordsHeight(e) {
        if (!recordsHeightResize) {
            return;
        }

        e.preventDefault();
        e.stopPropagation();
        recordsHeight -= e.movementY;
    }
</script>

<svelte:document
    on:mousemove={resizeRecordsHeight}
    on:mouseleave={() => (recordsHeightResize = false)}
    on:mouseup={() => (recordsHeightResize = false)}
/>

{#if showRecords}
    <ModalFooter class="p-0 d-block" style={"max-height:" + recordsHeight + "px"}>
        <div
            class="border-top border-dark border-2 d-flex m-0"
            style:cursor="ns-resize"
            on:mousedown|preventDefault={() => (recordsHeightResize = true)}
        ></div>
        <div
            class="m-0 d-flex justify-content-center"
            style:max-height="inherit"
            style:overflow-y="auto"
        >
            {#await getServiceRecords(origin, zoneId, service)}
                <div class="flex-fill d-flex justify-content-center">
                    <Spinner class="my-1" />
                </div>
            {:then serviceRecords}
                <TableRecords {serviceRecords} />
            {/await}
        </div>
    </ModalFooter>
{/if}
<ModalFooter>
    {#if update && origin && zoneId}
        <Button
            color="dark"
            outline={!showRecords}
            title={$t("domains.see-records")}
            on:click={() => (showRecords = !showRecords)}
        >
            <Icon name="code-square" />
        </Button>
    {/if}
    <div class="ms-auto"></div>
    {#if origin && zoneId && showRecords}
        <Label for="svc_ttl" title={$t("service.ttl-long")}>{$t("service.ttl")}</Label>
        <Input
            id="svc_ttl"
            min="0"
            type="number"
            style="width: 12%"
            title={$t("service.ttl-tip")}
            bind:value={service._ttl}
            on:input={(e) =>
                parseInt(e.target.value, 10)
                    ? (service._ttl = parseInt(e.target.value, 10))
                    : (service._ttl = 0)}
        />
    {:else if step === 2}
        <HelpButton color="info" href={helpHref} title={$t("common.help")} />
    {/if}
    {#if update}
        <Button
            color="danger"
            disabled={addServiceInProgress || deleteServiceInProgress || !canDelete}
            title={$t("service.delete")}
            on:click={() => dispatch("delete-service")}
        >
            {#if deleteServiceInProgress}
                <Spinner size="sm" />
            {:else}
                <Icon name="trash" />
            {/if}
        </Button>
    {/if}
    <Button color="secondary" on:click={toggle}>
        {$t("common.cancel")}
    </Button>
    {#if step === 2 && update}
        <Button
            disabled={addServiceInProgress || deleteServiceInProgress}
            {form}
            type="submit"
            color="success"
        >
            {#if addServiceInProgress}
                <Spinner label="Spinning" size="sm" />
            {/if}
            {$t("service.update")}
        </Button>
    {:else if step === 2}
        <Button {form} type="submit" color="primary">
            {$t("service.add")}
        </Button>
    {:else}
        <Button disabled={!canContinue} {form} type="submit" color="primary">
            {$t("common.continue")}
        </Button>
    {/if}
</ModalFooter>
