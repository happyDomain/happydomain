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
    import { page } from "$app/state";
    import type { ClassValue } from "svelte/elements";

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
    } from "@sveltestrap/sveltestrap";

    import { logout as APILogout, cleanUserSession } from "$lib/api/user";
    import HelpButton from "$lib/components/Help.svelte";
    import Logo from "$lib/components/Logo.svelte";
    import { appConfig, navigate } from "$lib/stores/config";
    import { userSession, refreshUserSession } from "$lib/stores/usersession";
    import { toasts } from "$lib/stores/toasts";
    import { t, locales, locale } from "$lib/translations";

    interface Props {
        class?: ClassValue;
        sw_state: { triedUpdate: boolean; hasUpdate: boolean };
    }

    let { class: className, sw_state }: Props = $props();

    function logout() {
        APILogout().then(
            () => {
                cleanUserSession();
                refreshUserSession().then(
                    () => {},
                    () => {
                        navigate("/login");
                    },
                );
            },
            (error) => {
                toasts.addErrorToast({
                    title: $t("errors.logout"),
                    message: error,
                    timeout: 20000,
                });
            },
        );
    }
</script>

<Navbar class="{className} py-1" id="nav" container expand="xs" light style="z-index: 100">
    <NavbarBrand
        href="/"
        class="me-3"
        style="padding: 0;"
        target={$userSession.id ? undefined : "_self"}
    >
        <Logo />
    </NavbarBrand>
    <Nav class="ms-auto align-items-center gap-1" navbar>
        <HelpButton size="sm" class="btn-icon" />
        {#if $userSession.id}
            <Dropdown nav inNavbar>
                <DropdownToggle nav caret class="user-toggle">
                    <Icon name="person-circle" />
                    {#if $userSession.email !== "_no_auth"}
                        <span class="d-inline d-sm-none">
                            {$userSession.email.split("@")[0]}
                        </span>
                    {/if}
                    <span class="d-none d-sm-inline">
                        {#if $userSession.email !== "_no_auth"}
                            {$userSession.email}
                        {:else}
                            {$t("menu.quick-menu")}
                        {/if}
                    </span>
                </DropdownToggle>
                <DropdownMenu end>
                    <DropdownItem
                        active={page.route && page.route.id == "/domains"}
                        href="/domains/"
                    >
                        <Icon name="globe2" class="me-2" />
                        {$t("menu.my-domains")}
                    </DropdownItem>
                    <DropdownItem
                        active={page.route && page.route.id == "/providers"}
                        href="/providers/"
                    >
                        <Icon name="hdd-network" class="me-2" />
                        {$t("menu.my-providers")}
                    </DropdownItem>
                    <DropdownItem divider />
                    <DropdownItem
                        active={page.route && page.route.id == "/providers/features"}
                        href="/providers/features"
                    >
                        <Icon name="buildings" class="me-2" />
                        {$t("menu.provider-features")}
                    </DropdownItem>
                    <DropdownItem
                        active={page.route && page.route.id == "/resolver/[[domain]]"}
                        href="/resolver"
                    >
                        <Icon name="search" class="me-2" />
                        {$t("menu.dns-resolver")}
                    </DropdownItem>
                    <DropdownItem divider />
                    <DropdownItem active={page.route && page.route.id == "/me"} href="/me">
                        <Icon name="gear" class="me-2" />
                        {$t("menu.my-account")}
                    </DropdownItem>
                    {#if $userSession.email !== "_no_auth"}
                        <DropdownItem divider />
                        <DropdownItem on:click={logout}>
                            <Icon name="box-arrow-right" class="me-2" />
                            {$t("menu.logout")}
                        </DropdownItem>
                    {/if}
                </DropdownMenu>
            </Dropdown>
        {:else}
            <Button class="btn-sm" color="info" outline href="/resolver">
                <Icon name="search" aria-hidden="true" />
                <span class="d-none d-sm-inline ms-1">{$t("menu.dns-resolver")}</span>
            </Button>

            {#if !$appConfig.disable_registration}
                <Button
                    class="d-none d-md-inline-block btn-sm"
                    outline={!page.route || page.route.id != "/register"}
                    color="dark"
                    href="/register"
                >
                    <Icon name="person-plus-fill" aria-hidden="true" />
                    <span class="ms-1">{$t("menu.signup")}</span>
                </Button>
            {/if}
            <Button
                class="d-none d-md-inline-block btn-sm"
                outline={!page.route || page.route.id != "/login"}
                color="primary"
                href="/login"
            >
                <Icon name="person-check-fill" aria-hidden="true" />
                <span class="ms-1">{$t("menu.signin")}</span>
            </Button>
            <Dropdown nav inNavbar>
                <DropdownToggle nav caret class="text-uppercase small">{$locale}</DropdownToggle>
                <DropdownMenu end>
                    {#each $locales as lang}
                        <DropdownItem active={$locale == lang} on:click={() => ($locale = lang)}>
                            {$t(`locales.${lang}`)}
                        </DropdownItem>
                    {/each}
                </DropdownMenu>
            </Dropdown>
        {/if}
    </Nav>
</Navbar>
