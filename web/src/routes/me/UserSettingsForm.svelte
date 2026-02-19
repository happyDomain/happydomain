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
    import { navigate } from "$lib/stores/config";

    import { Button, ButtonGroup, Icon, Input, Spinner } from "@sveltestrap/sveltestrap";

    import { saveAccountSettings } from "$lib/api/user";
    import type { UserSettings } from "$lib/model/usersettings";
    import { t, locales, locale } from "$lib/translations";
    import { refreshUserSession, userSession } from "$lib/stores/usersession";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        settings: UserSettings;
    }

    let { settings = $bindable() }: Props = $props();
    let formSent = $state(false);

    function saveSettings(e: SubmitEvent) {
        e.preventDefault();
        formSent = true;
        saveAccountSettings($userSession, settings).then(
            (settings) => {
                refreshUserSession().then(() => {
                    formSent = false;
                    if (settings.language != $locale) {
                        $locale = settings.language;
                    }

                    toasts.addToast({
                        title: $t("settings.success-change"),
                        message: $t("settings.success"),
                        timeout: 5000,
                        color: "success",
                    });

                    navigate("/");
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

<form class="list-group" onsubmit={saveSettings}>
    <div class="list-group-item">
        <div class="d-flex flex-column flex-md-row justify-content-md-between align-items-md-center gap-3">
            <label for="language-select" class="flex-grow-1 cursor-pointer mb-0">
                <div class="h5 mb-1">{$t("settings.language")}</div>
                <p class="mb-0 text-muted small">{$t("settings.language-description")}</p>
            </label>
            <div class="settings-input">
                <Input id="language-select" type="select" bind:value={settings.language}>
                    {#each $locales as lang}
                        <option value={lang}>{$t(`locales.${lang}`)}</option>
                    {/each}
                </Input>
            </div>
        </div>
    </div>

    <div class="list-group-item">
        <div class="d-flex flex-column flex-md-row justify-content-md-between align-items-md-center gap-3">
            <label for="fieldhint-select" class="flex-grow-1 cursor-pointer mb-0">
                <div class="h5 mb-1">{$t("settings.fieldhint.title")}</div>
                <p class="mb-0 text-muted small">{$t("settings.fieldhint.description")}</p>
            </label>
            <div class="settings-input">
                <Input id="fieldhint-select" type="select" bind:value={settings.fieldhint}>
                    <option value={0}>{$t("settings.fieldhint.hide")}</option>
                    <option value={1}>{$t("settings.fieldhint.tooltip")}</option>
                    <option value={2}>{$t("settings.fieldhint.focused")}</option>
                    <option value={3}>{$t("settings.fieldhint.always")}</option>
                </Input>
            </div>
        </div>
    </div>

    <div class="list-group-item">
        <div class="d-flex flex-column">
            <div class="mb-3">
                <h5 class="mb-1">{$t("settings.zoneview.title")}</h5>
                <p class="mb-0 text-muted small">{$t("settings.zoneview.description")}</p>
            </div>
            <ButtonGroup class="pt-2 pb-4" id="zoneview">
                <Button
                    type="button"
                    color="secondary"
                    outline={settings.zoneview !== 0}
                    on:click={() => (settings.zoneview = 0)}
                >
                    <Icon name="grid-fill" aria-hidden="true" /><br />
                    {$t("settings.zoneview.grid")}
                </Button>
                <Button
                    type="button"
                    color="secondary"
                    outline={settings.zoneview !== 1}
                    on:click={() => (settings.zoneview = 1)}
                >
                    <Icon name="list-ul" aria-hidden="true" /><br />
                    {$t("settings.zoneview.list")}
                </Button>
                <Button
                    type="button"
                    color="secondary"
                    outline={settings.zoneview !== 2}
                    on:click={() => (settings.zoneview = 2)}
                >
                    <Icon name="menu-button-wide-fill" aria-hidden="true" /><br />
                    {$t("settings.zoneview.records")}
                </Button>
            </ButtonGroup>
        </div>
    </div>

    <div class="list-group-item">
        <div class="d-flex flex-column flex-md-row justify-content-md-between align-items-start align-items-md-center gap-3">
            <label for="showrrtypes" class="flex-grow-1 cursor-pointer mb-0">
                <div class="h5 mb-1">{$t("settings.showrrtypes-title")}</div>
                <p class="mb-0 text-muted small">{$t("settings.showrrtypes")}</p>
            </label>
            <div>
                <div class="form-check form-switch">
                    <input
                        class="form-check-input"
                        type="checkbox"
                        role="switch"
                        id="showrrtypes"
                        bind:checked={settings.showrrtypes}
                    />
                    <label class="form-check-label visually-hidden" for="showrrtypes">{$t("settings.showrrtypes-title")}</label>
                </div>
            </div>
        </div>
    </div>

    <div class="list-group-item">
        <div class="d-flex justify-content-center justify-content-md-end align-items-center">
            <Button type="submit" color="primary" disabled={formSent} class="settings-input">
                {#if formSent}
                    <Spinner size="sm" class="me-2" />
                {/if}
                {$t("settings.save")}
            </Button>
        </div>
    </div>
</form>

<style>
    .settings-input {
        width: 100%;
    }

    @media (min-width: 768px) {
        .settings-input {
            width: 200px;
        }
    }
</style>
