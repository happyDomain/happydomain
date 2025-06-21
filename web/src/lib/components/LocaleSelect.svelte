<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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

    import { Input, Spinner } from "@sveltestrap/sveltestrap";

    import { saveAccountSettings } from "$lib/api/user";
    import type { UserSettings } from "$lib/model/usersettings";
    import { locales, locale } from "$lib/translations";
    import { refreshUserSession, userSession } from "$lib/stores/usersession";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    interface Props {
        [key: string]: any
    }

    let { ...rest }: Props = $props();

    let formSent = $state(false);

    let settings: UserSettings = $derived($userSession.settings);

    function saveLocale() {
        formSent = true;
        saveAccountSettings($userSession, settings).then(
            (settings) => {
                refreshUserSession().then(() => {
                    formSent = false;
                    if (settings.language != $locale) {
                        $locale = settings.language;
                    }
                });
            },
            (error) => {
                formSent = false;
                toasts.addErrorToast({
                    title: $t("errors.settings-change"),
                    message: error,
                    timeout: 10000,
                });
            },
        );
    }
</script>

<div class="d-flex gap-2 align-items-center">
    <Input
        id="locale-select"
        type="select"
        bind:value={settings.language}
        on:change={saveLocale}
        {...rest}
    >
        {#each $locales as lang}
            <option value={lang}>{$t(`locales.${lang}`)}</option>
        {/each}
    </Input>
    {#if formSent}
        <Spinner size="sm" />
    {/if}
</div>
