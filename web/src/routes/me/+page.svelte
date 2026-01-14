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
    import { goto } from "$app/navigation";

    import {
        Button,
        Container,
        Col,
        Input,
        InputGroup,
        ListGroup,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { userSession } from "$lib/stores/usersession";

    import ChangePasswordForm from "./ChangePasswordForm.svelte";
    import DeleteAccountCard from "./DeleteAccountCard.svelte";
    import SessionsManager from "./SessionsManager.svelte";
    import UserSettingsForm from "./UserSettingsForm.svelte";

    let is_auth_user_req = $userSession.id ? fetch(`/api/users/${$userSession.id}/is_auth_user`) : false;
</script>

<Container class="my-4 pb-5">
    <div class="text-center">
        <h1 class="display-6 fw-bold">
            {$t("settings.title")}
        </h1>
        <p class="lead mt-1" style="text-wrap: balance;">
            {$t("settings.subtitle")}
        </p>
    </div>
    {#if !$userSession.settings}
        <div class="d-flex justify-content-center">
            <Spinner color="primary" />
        </div>
    {:else}
        <h2 id="preferences" class="display-7 fw-bold mt-5">
            <i class="bi bi-sliders"></i> {$t("settings.preferences.title")}
        </h2>
        <p class="lead">
            {$t("settings.preferences.description")}
        </p>

        <UserSettingsForm bind:settings={$userSession.settings} />

        {#if $userSession.email !== "_no_auth"}
            <h2 id="security" class="display-7 fw-bold mt-5">
                <i class="bi bi-shield"></i> {$t("settings.security.title")}
            </h2>
            <p class="lead">
                {$t("settings.security.description")}
            </p>

            <SessionsManager />

            <h3 class="fw-bold mt-5" id="password-change">
                {$t("password.change")}
            </h3>
            <p>
                {$t("settings.security.password.description")}
            </p>
            {#await is_auth_user_req}
                <div class="d-flex justify-content-center my-2">
                    <Spinner />
                </div>
            {:then res}
                {#if res && res.status === 204}
                    <ChangePasswordForm />
                {:else}
                    {#await fetch("/auth/has_oidc") then res}
                        {#await res.json() then oidc}
                            <div class="alert alert-secondary">
                                {$t("account.no-password-change", { provider: oidc.provider })}
                            </div>
                        {/await}
                    {/await}
                {/if}
            {/await}

            <h2 class="display-7 fw-bold mt-5" id="delete-account">
                <i class="bi bi-x-circle"></i> {$t("account.delete.delete")}
            </h2>
            <p class="lead">
                {$t("account.delete.description")}
            </p>
            {#await is_auth_user_req}
                <div class="d-flex justify-content-center my-2">
                    <Spinner />
                </div>
            {:then res}
                <ListGroup>
                    <DeleteAccountCard externalAuth={res && res.status !== 204} />
                </ListGroup>
            {/await}
        {:else}
            <div class="alert alert-secondary mt-4">
                {$t("errors.account-no-auth")}
            </div>
        {/if}
    {/if}
</Container>
