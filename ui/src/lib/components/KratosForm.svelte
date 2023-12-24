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
 import { createEventDispatcher } from 'svelte';

 import {
     Alert,
     Button,
     Icon,
     Spinner,
     TabContent,
     TabPane,
 } from 'sveltestrap';

 import KratosFlow from '$lib/components/KratosFlow.svelte';
 import { t } from '$lib/translations';
 import { toasts } from '$lib/stores/toasts';
 import { refreshUserSession } from '$lib/stores/usersession';

 const dispatch = createEventDispatcher();

 export let flow: String;
 export let tabs = false;

 let form = { };
 let groups = [];
 let submissionInProgress = false;
 let error = null;
 let suggestLogout = false;
 let action_method = "";
 let action_url = "";

 async function getFlow(aal) {
     const res = await fetch(window.happydomain_ory_kratos_url + `self-service/${flow}/browser${aal ? '?aal=' + aal : ''}`,
         {
             method: "GET",
             headers: [
                 ["Accept", "application/json"],
                 ["Content-Type", "application/json"]
             ],
             credentials: 'include',
         }
     )
     suggestLogout = false;
     return {data: res, state: await res.json()};
 }

 async function treatForm({ data, next, state }) {
     if (next && data.status === 200) {
         if (state.session && (!state.session.active || !state.session.identity)) {
             state = (await getFlow('aal2')).state;
             suggestLogout = true;
         } else {
             dispatch('success', state);
         }
     }

     if (state.error && state.error.id === 'session_already_available') {
         state = (await getFlow('aal2')).state;
         suggestLogout = true;
     }

     if (state.error) {
         error = state.error;
         toasts.addErrorToast({
             title: state.error.message,
             message: state.error.reason,
             timeout: 30000,
         });
     } else {
         error = null;
     }

     if (state.ui) {
         const grps = [];

         for(const node of state.ui.nodes) {
             if (node.group !== "default" && !grps.includes(node.group)) {
                 grps.push(node.group);
             }
         }

         form = state;
         action_url = state.ui.action;
         action_method = state.ui.method;
         groups = grps;
     }

     submissionInProgress = false;
 }

 function submission(event) {
     submissionInProgress = true;
     fetch(action_url,
         {
             method: action_method,
             body: JSON.stringify(event.detail),
             headers: [
                 ["Accept", "application/json"],
                 ["Content-Type", "application/json"]
             ],
             credentials: 'include',
         }
     ).then(
         async (data) => ({ data, next: true, state: await data.json() })
     ).then(treatForm);
 }

 async function forceLogout() {
     await fetch(window.happydomain_ory_kratos_url + `self-service/logout/browser`,
                 {
                     method: "GET",
                     headers: [
                         ["Accept", "application/json"],
                         ["Content-Type", "application/json"]
                     ],
                     credentials: 'include',
                 }
     ).then(
         async (data) => data.json()
     ).then(
         async (state) => {
             await fetch(state.logout_url,
                         {
                             method: "GET",
                             headers: [
                                 ["Accept", "application/json"],
                                 ["Content-Type", "application/json"]
                             ],
                             credentials: 'include',
                         }
             );
         }
     );

     formreq = getFlow();
     formreq.then(treatForm);
}

 let formreq = getFlow();
 formreq.then(treatForm);
</script>

{#if error && error.message}
    <Alert color="danger">
        <Button
            class="float-end"
            color="link"
            size="sm"
            on:click={forceLogout}
        >
            <Icon
                name="door-open"
            />
        </Button>
        <strong>
            {error.message}.
        </strong>
        {error.reason}
        {#if error.details}
            <a href={error.details.docs} class="float-end" target="_blank">
                <Icon
                    name="info-circle-fill"
                    title={error.details.hint}
                />
            </a>
        {/if}
    </Alert>
{/if}
{#if form && form.ui}
    {#if form.ui.messages}
        {#each form.ui.messages as message}
            <Alert color={message.type === "error"?"danger":"info"}>
                <strong>
                    {message.text}
                </strong>
            </Alert>
        {/each}
    {/if}
    {#if form.ui.nodes}
        {#if tabs}
            <TabContent>
                {#each groups as group, ig}
                    <TabPane
                        class="pt-2"
                        tabId={group}
                               tab={group}
                        active={ig === 0}
                    >
                        <KratosFlow
                            {flow}
                            nodes={form.ui.nodes}
                            only={group}
                            {submissionInProgress}
                            on:submit={submission}
                        />
                    </TabPane>
                {/each}
            </TabContent>
        {:else}
            {#each groups as group, i}
                {#if i > 0}
                    <hr>
                {/if}

                <KratosFlow
                    {flow}
                    nodes={form.ui.nodes}
                    only={group}
                    {submissionInProgress}
                    on:submit={submission}
                />
            {/each}
        {/if}
    {/if}
{/if}
{#if suggestLogout}
    <div class="d-flex align-items-center justify-content-center">
        Quelque chose s'est mal passé&nbsp;?
        <Button
            color="link"
            type="button"
            on:click={forceLogout}
        >
            Déconnectez-vous
        </Button>
    </div>
{/if}
