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
    import { onMount } from "svelte";
    import { page } from "$app/state";
    import { fly, fade } from "svelte/transition";
    import { cubicOut } from "svelte/easing";

    import { Icon } from "@sveltestrap/sveltestrap";

    import { userSession } from "$lib/stores/usersession";
    import { t, locale } from "$lib/translations";

    let instancename = $state("");
    let showCard = $state(false);

    onMount(() => {
        instancename = encodeURIComponent(window.location.hostname);
    });
</script>

{#if showCard}
    <div
        role="presentation"
        class="vox-backdrop"
        transition:fade={{ duration: 200 }}
        onclick={() => (showCard = false)}
    ></div>
    <div
        class="card vox-card"
        transition:fly={{ x: 380, duration: 300, easing: cubicOut }}
    >
        <div class="card-body">
            <div class="row row-cols-2 g-2">
                <div class="col d-flex">
                    <a
                        href="https://matrix.to/#/#happyDNS:matrix.org"
                        target="_blank"
                        rel="noreferrer"
                        class="btn btn-outline-secondary flex-fill vox-action"
                        onclick={() => (showCard = false)}
                        data-umami-event="vox-people-chat"
                    >
                        <Icon name="chat-text" />
                        <span>Chat with us</span>
                    </a>
                </div>
                <div class="col d-flex">
                    <a
                        href="https://help.happydomain.org/{$locale}/"
                        target="_blank"
                        rel="noreferrer"
                        class="btn btn-outline-secondary flex-fill vox-action"
                        onclick={() => (showCard = false)}
                        data-umami-event="vox-people-help"
                    >
                        <Icon name="life-preserver" />
                        <span>Online help</span>
                    </a>
                </div>
                <div class="col d-flex">
                    <a
                        href="https://framaforms.org/quel-est-votre-avis-sur-happydns-1610366701?u={$userSession.id
                            ? $userSession.id
                            : 0}&amp;i={instancename}{page.route
                            ? '&p=' + page.route.id
                            : ''}&amp;l={$locale}"
                        target="_blank"
                        rel="noreferrer"
                        class="btn btn-outline-secondary flex-fill vox-action"
                        onclick={() => (showCard = false)}
                        data-umami-event="vox-people-feedback"
                    >
                        <Icon name="pen" />
                        <span>Write to us</span>
                    </a>
                </div>
                <div class="col d-flex">
                    <a
                        href="https://feedback.happydomain.org/"
                        target="_blank"
                        rel="noreferrer"
                        class="btn btn-outline-secondary flex-fill vox-action"
                        onclick={() => (showCard = false)}
                        data-umami-event="vox-people-feedback"
                    >
                        <Icon name="feather" />
                        <span>Give feedback</span>
                    </a>
                </div>
            </div>
        </div>
    </div>
{/if}
<button
    id="voxpeople"
    title={$t("common.survey")}
    class="d-flex btn justify-content-center align-items-center"
    class:vox-open={showCard}
    data-umami-event="vox-people"
    onclick={() => (showCard = !showCard)}
>
    <Icon name={showCard ? "x-lg" : "chat-right-text"} />
</button>

<style>
    .vox-backdrop {
        background-color: rgba(0, 0, 0, 0.3);
        backdrop-filter: blur(2px);
        position: fixed;
        inset: 0;
        z-index: 1050;
    }

    .vox-card {
        position: fixed;
        bottom: calc(7vh + 3.5rem);
        right: 1.5rem;
        z-index: 1052;
        max-width: 340px;
        width: calc(100vw - 3rem);
        border: none;
        border-radius: 1rem;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
        transform-origin: bottom right;
    }

    .vox-action {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.35rem;
        padding: 0.75rem 0.5rem;
        border-radius: 0.75rem;
        font-size: 0.95rem;
        font-weight: 500;
        transition: background-color 0.15s, border-color 0.15s;
    }

    .vox-action :global(.bi) {
        font-size: 1.4rem;
    }

    #voxpeople {
        position: fixed;
        bottom: 7vh;
        right: 1.5rem;
        z-index: 1051;
        height: 2.75rem;
        width: 2.75rem;
        border-radius: 50%;
        background: var(--bs-primary-bg-subtle);
        color: var(--bs-primary-text-emphasis);
        border: 1px solid var(--bs-primary-border-subtle);
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
        font-size: 1.1rem;
        transition: transform 0.2s ease, box-shadow 0.2s ease, filter 0.2s ease;
    }

    #voxpeople:hover {
        transform: scale(1.08);
        filter: brightness(0.95);
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
    }

    .vox-open {
        filter: brightness(0.9);
    }
</style>
