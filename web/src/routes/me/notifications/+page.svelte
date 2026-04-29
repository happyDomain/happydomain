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
    import { Container, TabContent, TabPane } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import { t } from "$lib/translations";

    import ChannelsManager from "./ChannelsManager.svelte";
    import HistoryList from "./HistoryList.svelte";
    import PreferencesManager from "./PreferencesManager.svelte";
</script>

<svelte:head>
    <title>{$t("settings.notifications.title")} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-4 pb-5">
    <PageTitle
        title={$t("settings.notifications.title")}
        subtitle={$t("settings.notifications.subtitle")}
    />

    <TabContent>
        <TabPane tabId="channels" tab={$t("settings.notifications.channels.title")} active>
            <div class="pt-3">
                <ChannelsManager />
            </div>
        </TabPane>
        <TabPane tabId="preferences" tab={$t("settings.notifications.preferences.title")}>
            <div class="pt-3">
                <PreferencesManager />
            </div>
        </TabPane>
        <TabPane tabId="history" tab={$t("settings.notifications.history.title")}>
            <div class="pt-3">
                <HistoryList />
            </div>
        </TabPane>
    </TabContent>
</Container>
