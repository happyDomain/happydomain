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
    import "../app.scss";
    import "bootstrap-icons/font/bootstrap-icons.css";

    import { goto } from "$app/navigation";
    import { page } from "$app/stores";

    import {
        Col,
        Container,
        Row,
        //Styles,
    } from "@sveltestrap/sveltestrap";

    import Header from "$lib/components/Header.svelte";
    import Logo from "$lib/components/Logo.svelte";
    import Toaster from "$lib/components/Toaster.svelte";
    import VoxPeople from "$lib/components/VoxPeople.svelte";
    import { appConfig } from "$lib/stores/config";
    import { providers } from "$lib/stores/providers";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    const { MODE } = import.meta.env;

    export let data: {
        route: { id: string | null };
        sw_state: { triedUpdate: boolean; hasUpdate: boolean };
    };

    window.onunhandledrejection = (e) => {
        if (e.reason.name == "NotAuthorizedError") {
            goto("/login");
            providers.set(null);
            toasts.addErrorToast({
                title: $t("errors.session.title"),
                message: $t("errors.session.content"),
                timeout: 10000,
            });
        } else {
            toasts.addErrorToast({
                message: e.reason.message,
                timeout: 30000,
            });
        }
    };

    let title = "happyDomain";
    $: if ($page.data.domain) {
        if (typeof $page.data.domain === "object") {
            title =
                $page.data.domain.domain.substring(0, $page.data.domain.domain.length - 1) +
                " - happyDomain";
        } else {
            title = $page.data.domain + " - happyDomain";
        }
    } else {
        title = "happyDomain";
    }
</script>

<svelte:head>
    <title>{title}</title>
</svelte:head>

<!--Styles /-->

{#if $appConfig.msg_header}
    <div
        class={($appConfig.msg_header.color ? "bg-" + $appConfig.msg_header.color : "bg-danger") +
            " text-light text-center fw-bolder pb-1"}
        id="msg_header"
        style="z-index: 101; margin-bottom: -.2em"
    >
        <small>
            {$appConfig.msg_header.text}
        </small>
    </div>
{/if}
<Header routeId={data.route.id} sw_state={data.sw_state} />

<div class="flex-fill d-flex flex-column justify-content-center bg-light">
    <slot></slot>
</div>

<Toaster />
{#if !$appConfig.hide_feedback && MODE == "production"}
    <VoxPeople routeId={data.route.id} />
{/if}
