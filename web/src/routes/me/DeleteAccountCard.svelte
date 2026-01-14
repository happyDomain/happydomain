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

<script module lang="ts">
    export const controls = {
        Open(): void { },
    };
</script>

<script lang="ts">
    import { goto } from "$app/navigation";

    import {
        Button,
        Input,
        ListGroupItem,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { deleteMyUser, deleteUserAccount } from "$lib/api/user";
    import { t } from "$lib/translations";
    import { userSession } from "$lib/stores/usersession";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        externalAuth?: boolean;
        [key: string]: any
    }

    let { externalAuth = false, ...rest }: Props = $props();

    let isOpen = $state(false);
    let password = $state("");
    let formSent = $state(false);

    function accountDeleted(): void {
        formSent = false;
        isOpen = false;
        toasts.addToast({
            title: $t("account.delete.deleted"),
            message: $t("account.delete.success"),
            type: "primary",
        });
        goto("/login");
    }

    function deletionError(err: Error): void {
        formSent = false;
        toasts.addErrorToast({
            title: $t("errors.account-delete"),
            message: err,
            timeout: 5000,
        });
    }

    function deleteMyAccount() {
        formSent = true;
        if (externalAuth) {
            deleteMyUser($userSession).then(accountDeleted, deletionError);
        } else {
            deleteUserAccount($userSession, password).then(accountDeleted, deletionError);
        }
    }

    function toggleModal(): void {
        isOpen = !isOpen;
    }

    function Open(): void {
        password = "";
        isOpen = true;
    }

    controls.Open = Open;
</script>

<ListGroupItem {...rest}>
    <div class="d-flex flex-column flex-md-row justify-content-md-between align-items-stretch align-items-md-center gap-3">
        <div>
            <p class="mb-0">
                {$t("account.delete.confirm")}
            </p>
            <p class="mb-0 text-muted" style="line-height: 1.1">
                <small>
                    {$t("account.delete.consequence")}
                </small>
            </p>
        </div>
        <button
            type="button"
            class="btn btn-danger delete-button"
            disabled={formSent}
            onclick={Open}
        >
            {#if formSent}
                <Spinner size="sm" class="me-2" />
            {/if}
            {$t("account.delete.delete-button")}
        </button>
    </div>
</ListGroupItem>

{#if externalAuth}
    <Modal isOpen={isOpen} toggle={toggleModal}>
        <ModalHeader toggle={toggleModal}>
            {$t("account.delete.delete")}
        </ModalHeader>
        <ModalBody>
            <p>
                {$t("account.delete.confirm-twice")}
            </p>
            <p class="text-muted" style="line-height: 1.1">
                <small>
                    {$t("account.delete.remain-data")}
                </small>
            </p>
        </ModalBody>
        <ModalFooter>
            <Button color="danger" on:click={deleteMyAccount}>
                {$t("account.delete.delete")}
            </Button>
            <Button
                color="secondary"
                on:click={() => (isOpen = !isOpen)}
            >
                {$t("common.cancel")}
            </Button>
        </ModalFooter>
    </Modal>
{:else}
    <Modal isOpen={isOpen} toggle={toggleModal}>
        <ModalHeader toggle={toggleModal}>
            {$t("account.delete.delete")}
        </ModalHeader>
        <ModalBody>
            <p>
                {$t("account.delete.confirm-twice")}
            </p>
            <div>
                <label for="currentPassword-forDeletion">
                    {$t("account.delete.confirm-password")}
                </label>
                <Input
                    id="currentPassword-forDeletion"
                    class="border-danger"
                    autocomplete="off"
                    autofocus
                    required
                    placeholder="xXxXxXxXxX"
                    type="password"
                    bind:value={password}
                />
            </div>
            <p class="text-muted" style="line-height: 1.1">
                <small>
                    {$t("account.delete.remain-data")}
                </small>
            </p>
        </ModalBody>
        <ModalFooter>
            <Button color="danger" on:click={deleteMyAccount}>
                {$t("account.delete.delete")}
            </Button>
            <Button
                color="secondary"
                on:click={() => (isOpen = !isOpen)}
            >
                {$t("common.cancel")}
            </Button>
        </ModalFooter>
    </Modal>
{/if}

<style>
    .delete-button {
        width: 100%;
    }

    @media (min-width: 768px) {
        .delete-button {
            width: auto;
        }
    }
</style>
