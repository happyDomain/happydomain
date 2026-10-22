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
    import { resolve } from "$app/paths";
    import { onMount } from "svelte";
    import "bootstrap-icons/font/bootstrap-icons.css";
    import "@fontsource-variable/lexend";
    import "@fontsource-variable/roboto";
    import "@fontsource-variable/jetbrains-mono";
    import "@fontsource-variable/jetbrains-mono/wght-italic.css";
    import "../app.scss";

    import Header from "$lib/components/Header.svelte";
    import NavigationProgress from "$lib/components/NavigationProgress.svelte";
    import ReportProblem from "$lib/components/modals/ReportProblem.svelte";
    import UnlockKeystore from "$lib/components/modals/UnlockKeystore.svelte";
    import Toaster from "$lib/components/Toaster.svelte";
    import VoxPeople from "$lib/components/VoxPeople.svelte";
    import { appConfig } from "$lib/stores/config";
    import { providers } from "$lib/stores/providers";
    import { registerReportModal } from "$lib/stores/report";
    import { toasts } from "$lib/stores/toasts";
    import { locale, t } from "$lib/translations";

    $effect(() => {
        document.documentElement.lang = $locale;
    });

    const { MODE } = import.meta.env;

    let {
        children,
    }: {
        children?: import("svelte").Snippet;
    } = $props();

    let reportModal: ReportProblem | undefined = $state();

    $effect(() => {
        if (reportModal) registerReportModal((prefill?: string) => reportModal?.open(prefill));
    });

    onMount(() => {
        // Nothing is rendered until every load of the initial route resolves:
        // reaching this point means the app is ready to take over the splash.
        document.getElementById("app-splash")?.remove();

        window.onunhandledrejection = (e) => {
            if (e.reason.name == "NotAuthorizedError") {
                goto(resolve("/login"));
                providers.set(undefined);
                toasts.addErrorToast({
                    title: $t("errors.session.title"),
                    message: $t("errors.session.content"),
                    timeout: 10000,
                });
            } else if (e.reason.name == "KeystoreLockedError") {
                // The user chose not to unlock it: no bug to report.
                toasts.addErrorToast({
                    message: $t("keystores.still-locked"),
                    timeout: 10000,
                });
            } else {
                // Nothing caught this one: whatever it is, we didn't plan for
                // it, so let the user hand it over to us.
                toasts.addErrorToast({
                    message: e.reason.message,
                    timeout: 30000,
                    reportable: true,
                });
            }
        };
    });
</script>

{#if $appConfig.msg_header}
    <div
        class={"text-center fw-semibold py-1 " +
            ($appConfig.msg_header.color
                ? "text-bg-" + $appConfig.msg_header.color
                : "text-bg-danger")}
        id="msg_header"
        style="z-index: 101"
    >
        <small>
            {$appConfig.msg_header.text}
        </small>
    </div>
{/if}
<NavigationProgress />
<Header />

<main class="flex-fill d-flex flex-column justify-content-center bg-body">
    {@render children?.()}
</main>

<Toaster />
<ReportProblem bind:this={reportModal} />
<UnlockKeystore />
{#if !$appConfig.hide_feedback && MODE == "production"}
    <VoxPeople />
{/if}
