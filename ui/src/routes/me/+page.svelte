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
 import { goto } from '$app/navigation';

 import {
     Button,
     Card,
     CardBody,
     Container,
     Col,
     Input,
     InputGroup,
     ListGroup,
     ListGroupItem,
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
     Row,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import { addSession, deleteSession, deleteSessions, getCurrentSession, listSessions } from '$lib/api/sessions';
 import ChangePasswordForm from '$lib/components/ChangePasswordForm.svelte';
 import DeleteAccountCard from '$lib/components/DeleteAccountCard.svelte';
 import UserSettingsForm from '$lib/components/UserSettingsForm.svelte';
 import type { Session } from '$lib/model/session';
 import { t } from '$lib/translations';
 import { userSession } from '$lib/stores/usersession';

 let is_auth_user_req = fetch(`/api/users/${$userSession.id}/is_auth_user`);
 let current_session_req = getCurrentSession();
 let sessions_req = listSessions();

 async function del(session: Session) {
     session.delete_in_progress = true;
     await deleteSession(session.id);
     sessions_req = listSessions();
 }

 let is_closing_sessions = false;
 async function closeSessions() {
     is_closing_sessions = true;
     await deleteSessions();
     is_closing_sessions = false;
     goto('/login');
 }

 let newSessionModalOpen = false;
 let newSessionDescription = "";
 $: if (newSessionModalOpen) {
     newSessionSecret = "";
     newSessionDescription = "";
     newSessionSecretShown = false;
 }
 let creating_session_in_progress = false;
 let newSessionSecret = "";
 let newSessionSecretShown = false;
 async function createSession() {
     creating_session_in_progress = true;

     const session = await addSession(newSessionDescription);
     newSessionSecret = session.id;

     creating_session_in_progress = false;
     newSessionSecretShown = false;
     sessions_req = listSessions();
 }

 let secretCopiedToClipboard = false;
 async function copySecretToClipboard() {
     secretCopiedToClipboard = null;
     await navigator.clipboard.writeText(newSessionSecret);
     secretCopiedToClipboard = true;
 }
</script>

<Container class="my-4">
    <h2 id="settings">
        {$t('settings.title')}
    </h2>
    {#if !$userSession}
        <div class="d-flex justify-content-center">
            <Spinner color="primary" />
        </div>
    {:else}
        <Row>
            {#if $userSession.settings}
                <Card class="offset-md-2 col-8">
                    <CardBody>
                        <UserSettingsForm settings={$userSession.settings} />
                    </CardBody>
                </Card>
            {/if}
        </Row>

        <hr>
        <div class="d-flex justify-content-between">
            <h2 id="sessions">
                {$t('settings.sessions')}
            </h2>
            <div>
                <Button
                    color="info"
                    outline
                    on:click={() => newSessionModalOpen = true}
                >
                    <i class="bi bi-plus-lg" />
                    {$t('sessions.create')}
                </Button>
                <Button
                    color="danger"
                    disabled={is_closing_sessions}
                    outline
                    on:click={closeSessions}
                >
                    {#if is_closing_sessions}
                        <Spinner size="sm" />
                    {:else}
                        <i class="bi bi-door-open" />
                    {/if}
                    {$t('sessions.close-all')}
                </Button>
            </div>
        </div>
        {#await current_session_req}
            <div class="d-flex justify-content-center my-2">
                <Spinner />
            </div>
        {:then current_session}
            {#await sessions_req}
                <div class="d-flex justify-content-center my-2">
                    <Spinner />
                </div>
            {:then sessions}
                <ListGroup>
                    {#each sessions as session (session.id)}
                        <ListGroupItem class="d-flex align-items-center">
                            <div class="flex-fill">
                                <div class="text-truncate">
                                    {session.description}
                                    <small class="text-muted">
                                        {#await window.crypto.subtle.digest("SHA-1", new TextEncoder().encode(session.id)) then sessid}
                                            {Array.from(new Uint8Array(sessid)).map((b) => b.toString(16).padStart(2, "0")).join("")}
                                        {/await}
                                    </small>
                                    {#if session.id === current_session.id}
                                        ({$t('sessions.current')})
                                    {/if}
                                </div>
                                <div>
                                    {$t('history.created-on')}
                                    {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "medium"}).format(new Date(session.time))}
                                    &#9679;
                                    {$t('history.used-on')}
                                    {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "medium"}).format(new Date(session.upd))}
                                    &#9679;
                                    {$t('history.expires-on')}
                                    {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "medium"}).format(new Date(session.exp))}
                                </div>
                            </div>
                            <div>
                                <Button
                                    color="danger"
                                    disabled={session.id === current_session.id || session.delete_in_progress || is_closing_sessions}
                                    outline
                                    on:click={() => del(session)}
                                >
                                    {#if session.delete_in_progress}
                                        <Spinner size="sm" />
                                    {:else}
                                        <i class="bi bi-trash" />
                                    {/if}
                                </Button>
                            </div>
                        </ListGroupItem>
                    {/each}
                </ListGroup>
            {/await}
        {/await}
        {#if $userSession.email !== '_no_auth'}
            <hr>
            <h2 id="password-change">
                {$t('password.change')}
            </h2>
            {#await is_auth_user_req}
                <div class="d-flex justify-content-center my-2">
                    <Spinner />
                </div>
            {:then res}
                {#if res.status === 204}
                    <Row>
                        <Col md={{size: 8, offset: 2}}>
                        <Card>
                            <CardBody>
                                <ChangePasswordForm />
                            </CardBody>
                        </Card>
                        </Col>
                    </Row>
                {:else}
                    {#await fetch('/auth/has_oidc') then res}
                        {#await res.json() then oidc}
                            <div class="m-5 alert alert-secondary">
                                {$t('account.no-password-change', {provider: oidc.provider})}
                            </div>
                        {/await}
                    {/await}
                {/if}
            {/await}
            <hr>
            <h2 id="delete-account">
                {$t('account.delete.delete')}
            </h2>
            {#await is_auth_user_req}
                <div class="d-flex justify-content-center my-2">
                    <Spinner />
                </div>
            {:then res}
                <Row>
                    <Col md={{size: 8, offset: 2}}>
                    <DeleteAccountCard
                        externalAuth={res.status !== 204}
                    />
                    </Col>
                </Row>
            {/await}
        {:else}
            <div class="m-5 alert alert-secondary">
                {$t('errors.account-no-auth')}
            </div>
        {/if}
    {/if}
</Container>

<Modal
    isOpen={newSessionModalOpen}
    toggle={() => newSessionModalOpen = !newSessionModalOpen}
>
    <ModalHeader
        toggle={() => newSessionModalOpen = !newSessionModalOpen}
    >
        {$t('sessions.create')}
    </ModalHeader>
    <ModalBody>
        {#if newSessionSecret === ""}
            <form
                on:submit={createSession}
            >
                <label for="session-description">
                    {$t('sessions.description')}
                </label>
                <Input
                    id="session-description"
                    autofocus
                    required
                    placeholder={`${navigator.appName} on ${navigator.platform}`}
                    bind:value={newSessionDescription}
                />
            </form>
        {:else}
            <div>
                <label for="session-secret">
                    {$t('sessions.secret')}
                </label>
                <InputGroup>
                    <Input
                        id="session-secret"
                        readonly
                        type={newSessionSecretShown ? "text" : "password"}
                        value={newSessionSecret}
                    />
                    <Button
                        color="info"
                        outline
                        on:click={() => newSessionSecretShown = !newSessionSecretShown}
                    >
                        <i class="bi bi-eye" />
                    </Button>
                    <Button
                        color="info"
                        on:click={copySecretToClipboard}
                    >
                        {#if secretCopiedToClipboard === null}
                            <Spinner size="sm" />
                        {:else if secretCopiedToClipboard}
                            <i class="bi bi-clipboard-check" />
                        {:else}
                            <i class="bi bi-clipboard" />
                        {/if}
                    </Button>
                </InputGroup>
            </div>
        {/if}
        <hr class="mt-4 mb-3">
        <p>
            {$t('sessions.usage')}
        </p>
        <pre>curl -H "Authorization: Bearer {newSessionSecretShown ? newSessionSecret : "SaMpLeSeCrEt"}" {window.location.origin}/api/domains

</pre>
    </ModalBody>
    <ModalFooter>
        {#if newSessionSecret === ""}
            <Button
                color="primary"
                disabled={creating_session_in_progress}
                on:click={createSession}
            >
                {#if creating_session_in_progress}
                    <Spinner size="sm" />
                {:else}
                    <i class="bi bi-plus-lg" />
                {/if}
                {$t('sessions.create')}
            </Button>
            <Button
                color="secondary"
                on:click={() => newSessionModalOpen = !newSessionModalOpen}
            >
                {$t('common.cancel')}
            </Button>
        {:else}
            <Button
                color="primary"
                on:click={() => newSessionModalOpen = !newSessionModalOpen}
            >
                {$t('common.got-it')}
            </Button>
        {/if}
    </ModalFooter>
</Modal>
