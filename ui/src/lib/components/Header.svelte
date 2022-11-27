<script>
 import { goto } from '$app/navigation';

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
 import Logo from '$lib/components/Logo.svelte';
 import { userSession, refreshUserSession } from '$lib/stores/usersession';
 import { config as tsConfig, t, locales, locale } from '$lib/translations';

 export { className as class };
 let className = '';

 function logout() {
     APILogout().then(
         () => {
             refreshUserSession().then(
                 () => { },
                 () => {
                     goto('/');
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
                outline
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
