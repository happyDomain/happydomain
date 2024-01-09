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
 import { page } from '$app/stores'

 import {
     Button,
     Dropdown,
     DropdownItem,
     DropdownMenu,
     DropdownToggle,
     Icon,
     Navbar,
     NavbarBrand,
     Nav,
 } from '@sveltestrap/sveltestrap';

 import { logout as APILogout } from '$lib/api/user';
 import HelpButton from '$lib/components/Help.svelte';
 import Logo from '$lib/components/Logo.svelte';
 import { providersSpecs } from '$lib/stores/providers';
 import { userSession, refreshUserSession } from '$lib/stores/usersession';
 import { toasts } from '$lib/stores/toasts';
 import { t, locales, locale } from '$lib/translations';

 export { className as class };
 let className = '';

 export let routeId: string | null;
 export let sw_state: {triedUpdate: boolean; hasUpdate: boolean;};
 let helpLink = "";
 $: if (routeId && routeId.startsWith("/providers/new/[ptype]")) {
     helpLink = getHelpPathFromProvider($page.url.pathname.split("/")[3]);
 } else if (routeId) {
     helpLink = 'https://help.happydomain.org/' + encodeURIComponent($locale) + getHelpPathFromRoute(routeId);
 } else {
     helpLink = 'https://help.happydomain.org/' + encodeURIComponent($locale);
 }

 function getHelpPathFromProvider(ptype: string): string {
     if ($providersSpecs && $providersSpecs[ptype]) {
         return $providersSpecs[ptype].helplink;
     } else {
         return 'https://help.happydomain.org/';
     }
 }

 function getHelpPathFromRoute(routeId: string | null) {
     if (routeId === null) return "/";

     const path = routeId.split("/");

     if (path.length < 2) return "/";

     switch(path[1]) {
         case "":
             return "/pages/home/";
         case "providers":
             if (path.length > 2) {
                 if (path[2] == "new") return "/pages/source-new-choice/";
                 return "/pages/source-update/";
             }
             return "/pages/source-list/";
         case "domains":
             if (path.length == 2) return "/pages/home/";
             if (path.length > 3 && path[3] == "new") return "/pages/domain-new/";
             return "/pages/domain-abstract/";
         case "me":
             return "/pages/me/";
         case "resolver":
             return "/pages/tools-client/";
         default:
             return "/";
     }
 }

 let activemenu = "";
 $: {
     const path = $page.url.pathname.split("/");
     if (path.length > 1) {
         activemenu = path[1];
     }
 }

 function logout() {
     APILogout().then(
         () => {
             refreshUserSession().then(
                 () => { },
                 () => {
                     goto('/login');
                 }
             )
         },
         (error) => {
             toasts.addErrorToast({
                 title: $t('errors.logout'),
                 message: error,
                 timeout: 20000,
             })
         }
     )
 }
</script>

<Navbar
    class="{className} {$userSession?'p-0':''}"
    style="z-index: 100"
    container
    expand="xs"
    light
>
    <NavbarBrand
        href="/"
        style="padding: 0; margin: -.5rem 0;"
        target={$userSession?undefined:"_self"}
    >
        <Logo />
    </NavbarBrand>
    <Nav class="ms-auto align-items-center" navbar>
        <HelpButton
            href={helpLink}
            size={$userSession?"sm":undefined}
            class={$userSession?"my-2":"mx-1"}
        />
        {#if $userSession}
            <Dropdown nav inNavbar>
                <DropdownToggle nav caret>
                    <Button
                        color="dark"
                        size="sm"
                    >
                        <Icon name="person" />
                        {#if $userSession.email !== '_no_auth'}
                            {$userSession.email}
                        {:else}
                            {$t('menu.quick-menu')}
                        {/if}
                    </Button>
                </DropdownToggle>
                <DropdownMenu end>
                    <DropdownItem href="/domains/">
                        {$t('menu.my-domains')}
                    </DropdownItem>
                    <DropdownItem href="/providers/">
                        {$t('menu.my-providers')}
                    </DropdownItem>
                    <DropdownItem divider />
                    <DropdownItem href="/providers/features">
                        {$t('menu.provider-features')}
                    </DropdownItem>
                    <DropdownItem href="/resolver">
                        {$t('menu.dns-resolver')}
                    </DropdownItem>
                    <DropdownItem divider />
                    <DropdownItem href="/me">
                        {$t('menu.my-account')}
                    </DropdownItem>
                    {#if $userSession.email !== '_no_auth'}
                        <DropdownItem divider />
                        <DropdownItem on:click={logout}>
                            {$t('menu.logout')}
                        </DropdownItem>
                    {/if}
                </DropdownMenu>
            </Dropdown>
        {:else}
            <Button
                class="mx-1"
                color="info"
                href="/resolver"
            >
                <Icon
                    name="list"
                    aria-hidden="true"
                />
                {$t('menu.dns-resolver')}
            </Button>

            <Button
                class="d-none d-md-inline-block mx-1"
                outline={activemenu != "join"}
                color="dark"
                href="/join"
            >
                <Icon
                    name="person-plus-fill"
                    aria-hidden="true"
                />
                {$t('menu.signup')}
            </Button>
            <Button
                class="d-none d-md-inline-block mx-1"
                outline={activemenu == "join"}
                color="primary"
                href="/login"
            >
                <Icon
                    name="person-check-fill"
                    aria-hidden="true"
                />
                {$t('menu.signin')}
            </Button>
            <Dropdown nav inNavbar>
                <DropdownToggle nav caret>{$locale}</DropdownToggle>
                <DropdownMenu end>
                    {#each $locales as lang}
                        <DropdownItem on:click={() => $locale = lang}>
                            {lang}
                        </DropdownItem>
                    {/each}
                </DropdownMenu>
            </Dropdown>
        {/if}
    </Nav>
</Navbar>
