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
 } from 'sveltestrap';

 import { logout as APILogout } from '$lib/api/user';
 import HelpButton from '$lib/components/Help.svelte';
 import Logo from '$lib/components/Logo.svelte';
 import { userSession, refreshUserSession } from '$lib/stores/usersession';
 import { toasts } from '$lib/stores/toasts';
 import { t, locales, locale } from '$lib/translations';

 export { className as class };
 let className = '';

 export let routeId: string | null;
 let helpLink = "";
 $: helpLink = 'https://help.happydomain.org/' + encodeURIComponent($locale) + getHelpPathFromRoute(routeId);

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
    container
    expand="xs"
    light
>
    <NavbarBrand
        href="/"
        style="padding: 0; margin: -.5rem 0;"
    >
        <Logo />
    </NavbarBrand>
    <Nav class="ms-auto" navbar>
        <HelpButton
            href={helpLink}
            size={$userSession?"sm":undefined}
            class={$userSession?"my-2":"me-2"}
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
                class="me-2"
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
                outline={activemenu == "join"}
                color="primary"
                class="ms-2"
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
