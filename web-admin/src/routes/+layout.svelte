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
    import "../app.scss";
    import "bootstrap-icons/font/bootstrap-icons.css";

    import { goto } from "$app/navigation";
    import { page } from "$app/state";

    import {
        Badge,
        Collapse,
        Nav,
        Navbar,
        NavbarToggler,
        NavbarBrand,
        NavItem,
        NavLink,
    } from "@sveltestrap/sveltestrap";

    import Logo from "$lib/components/Logo.svelte";
    import Toaster from "$lib/components/Toaster.svelte";
    import { appConfig } from "$lib/stores/config";
    import { providers } from "$lib/stores/providers";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    const { MODE } = import.meta.env;

    let { children }: {
        children?: import('svelte').Snippet;
    } = $props();

    window.onunhandledrejection = (e) => {
        toasts.addErrorToast({
            message: e.reason.message,
            timeout: 30000,
        });
    };

    let isOpen = $state(false);

    function handleUpdate(event: CustomEvent<boolean>) {
        isOpen = event.detail;
    }
</script>

<Navbar
    style="z-index: 100"
    container="md"
    expand="md"
    color="light"
>
    <NavbarBrand
        href="/"
        style="padding: 0; margin: -.5rem 1rem -.5rem 0;"
        target="_self"
    >
        <Logo />
        <Badge color="danger" class="d-none d-sm-inline">ADMIN</Badge>
    </NavbarBrand>
    <NavbarToggler on:click={() => (isOpen = !isOpen)} />
    <Collapse {isOpen} navbar expand="md" on:update={handleUpdate}>
        <Nav class="ms-auto align-items-center" navbar>
            <NavItem>
                <NavLink href="/" active={page && page.url.pathname == '/'}>Dashboard</NavLink>
            </NavItem>
            <NavItem>
                <NavLink href="/auth_users" active={page && page.url.pathname.startsWith('/auth_users')}>Auth</NavLink>
            </NavItem>
            <NavItem>
                <NavLink href="/users" active={page && page.url.pathname.startsWith('/users')}>Users</NavLink>
            </NavItem>
            <NavItem>
                <NavLink href="/domains" active={page && page.url.pathname.startsWith('/domains')}>Domains</NavLink>
            </NavItem>
            <NavItem>
                <NavLink href="/providers" active={page && page.url.pathname.startsWith('/providers')}>Providers</NavLink>
            </NavItem>
            <NavItem>
                <NavLink href="/sessions" active={page && page.url.pathname.startsWith('/sessions')}>Sessions</NavLink>
            </NavItem>
        </Nav>
    </Collapse>
</Navbar>

<main class="flex-fill d-flex flex-column justify-content-center bg-light">
    {@render children?.()}
</main>

<Toaster />
