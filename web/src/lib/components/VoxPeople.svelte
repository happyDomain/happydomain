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
    import { Icon } from "@sveltestrap/sveltestrap";

    const { MODE } = import.meta.env;

    import { userSession } from "$lib/stores/usersession";
    import { t, locale } from "$lib/translations";

    export let routeId: string | null;
    const instancename = encodeURIComponent(window.location.hostname);
    let showCard = false;
</script>

{#if MODE == "production"}
    {#if showCard}
        <div
            style="background-color: #0007; position: fixed; width: 100vw; height: 100vh; top:0; left: 0; z-index: 1050"
            on:click={() => (showCard = false)}
        />
        <div
            class="card"
            style="position: fixed; bottom: calc(7vh + max(1.7vw, 1.7vh)); right: calc(4vw + max(1.7vw, 1.7vh)); z-index: 1052; max-width: 400px;"
        >
            <div class="card-body row row-cols-2 justify-content-center align-items-center">
                <div class="col d-flex mb-3 flex-fill">
                    <a
                        href="https://matrix.to/#/#happyDNS:matrix.org"
                        target="_blank"
                        rel="noreferrer"
                        class="btn btn-lg btn-light flex-fill"
                        on:click={() => (showCard = false)}
                        data-umami-event="vox-people-chat"
                    >
                        <Icon name="chat-text" /><br />
                        <small>Chat with us</small>
                    </a>
                </div>
                <div class="col d-flex mb-3">
                    <a
                        href="https://help.happydomain.org/{$locale}/"
                        target="_blank"
                        rel="noreferrer"
                        class="btn btn-lg btn-light flex-fill"
                        on:click={() => (showCard = false)}
                        data-umami-event="vox-people-help"
                    >
                        <Icon name="life-preserver" /><br />
                        <small>Online help</small>
                    </a>
                </div>
                <div class="col d-flex flex-fill">
                    <a
                        href="https://framaforms.org/quel-est-votre-avis-sur-happydns-1610366701?u={$userSession
                            ? $userSession.id
                            : 0}&amp;i={instancename}&amp;p={routeId}&amp;l={$locale}"
                        target="_blank"
                        rel="noreferrer"
                        class="btn btn-lg btn-light flex-fill fw-bolder"
                        on:click={() => (showCard = false)}
                        data-umami-event="vox-people-feedback"
                    >
                        <Icon name="pen" /><br />
                        <small>Write to us</small>
                    </a>
                </div>
                <div class="col d-flex">
                    <a
                        href="https://feedback.happydomain.org/"
                        target="_blank"
                        rel="noreferrer"
                        class="btn btn-lg btn-light flex-fill fw-bolder"
                        on:click={() => (showCard = false)}
                        data-umami-event="vox-people-feedback"
                    >
                        <Icon name="feather" /><br />
                        <small>Give your feedback</small>
                    </a>
                </div>
            </div>
        </div>
    {/if}
    <button
        id="voxpeople"
        title={$t("common.survey")}
        class="d-flex btn btn-light justify-content-center align-items-center"
        data-umami-event="vox-people"
        on:click={() => (showCard = !showCard)}
    >
        <Icon name="chat-right-text" />
    </button>
{/if}

<style>
    #voxpeople {
        position: fixed;
        bottom: 7vh;
        right: 4vw;
        z-index: 1051;
        height: max(4vw, 4vh);
        width: max(4vw, 4vh);
        border-radius: 4vw;
        box-shadow: 0 0px 3px 0 #9332bb;
        font-size: max(1.7vw, 1.7vh);
    }
</style>
