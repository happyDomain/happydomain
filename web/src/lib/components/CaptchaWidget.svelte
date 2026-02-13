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
    import { onMount } from "svelte";
    import { appConfig } from "$lib/stores/config";

    let { token = $bindable() }: { token: string | null } = $props();

    let container: HTMLDivElement | undefined = $state();
    let widgetId: unknown = $state(undefined);

    const provider = $derived($appConfig.captcha_provider);
    const siteKey = $derived($appConfig.captcha_site_key ?? "");

    function onToken(t: string) {
        token = t;
    }

    function loadScript(src: string): Promise<void> {
        return new Promise((resolve, reject) => {
            if (document.querySelector(`script[src="${src}"]`)) {
                resolve();
                return;
            }
            const script = document.createElement("script");
            script.src = src;
            script.async = true;
            script.defer = true;
            script.onload = () => resolve();
            script.onerror = reject;
            document.head.appendChild(script);
        });
    }

    async function renderWidget() {
        if (!container || !provider || !siteKey) return;

        if (provider === "hcaptcha") {
            await loadScript("https://js.hcaptcha.com/1/api.js?render=explicit");
            // @ts-ignore
            widgetId = hcaptcha.render(container, { sitekey: siteKey, callback: onToken });
        } else if (provider === "recaptchav2") {
            await loadScript("https://www.google.com/recaptcha/api.js?render=explicit");
            // @ts-ignore
            widgetId = grecaptcha.render(container, { sitekey: siteKey, callback: onToken });
        } else if (provider === "turnstile") {
            await loadScript(
                "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit",
            );
            // @ts-ignore
            widgetId = turnstile.render(container, { sitekey: siteKey, callback: onToken });
        }
    }

    export function reset() {
        token = null;
        if (widgetId === undefined) return;

        if (provider === "hcaptcha") {
            // @ts-ignore
            hcaptcha.reset(widgetId);
        } else if (provider === "recaptchav2") {
            // @ts-ignore
            grecaptcha.reset(widgetId);
        } else if (provider === "turnstile") {
            // @ts-ignore
            turnstile.reset(widgetId);
        }
    }

    onMount(() => {
        renderWidget();
    });
</script>

{#if provider}
    <div bind:this={container} class="my-2"></div>
{/if}
