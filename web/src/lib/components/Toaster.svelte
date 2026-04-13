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
    import { Toast, ToastHeader } from "@sveltestrap/sveltestrap";

    import { toasts } from "$lib/stores/toasts";

    let hidden = false;

    onMount(() => {
        const handleVisibilityChange = () => {
            hidden = document.hidden;
            for (const toast of $toasts) {
                if (document.hidden) {
                    toast.pause();
                } else {
                    toast.resume();
                }
            }
        };

        document.addEventListener("visibilitychange", handleVisibilityChange);
        return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
    });
</script>

<div class="toast-container position-fixed top-0 end-0 p-3 mt-5" class:page-hidden={hidden} style="z-index: 1060">
    {#each $toasts as toast}
        <Toast onmouseenter={() => toast.pause()} onmouseleave={() => toast.resume()}>
            <ToastHeader toggle={() => toast.dismiss()} icon={toast.getColor()}>
                {#if toast.title}{toast.title}{:else}happyDomain{/if}
            </ToastHeader>
            {#if toast.timeout}
                <div
                    class="toast-progress bg-{toast.getColor()}"
                    style="animation-duration: {toast.timeout}ms"
                ></div>
            {/if}
            <div
                class="toast-body"
                role="button"
                tabindex="0"
                style={toast.onclick ? "cursor: pointer" : ""}
                onclick={() => {
                    if (toast.onclick) toast.onclick();
                }}
                onkeydown={(e) => {
                    if ((e.key === "Enter" || e.key === " ") && toast.onclick) {
                        e.preventDefault();
                        toast.onclick();
                    }
                }}
            >
                {toast.message}
            </div>
        </Toast>
    {/each}
</div>

<style>
    .toast-container :global(.toast) {
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
        border: none;
        border-radius: 0.5rem;
        overflow: hidden;
    }

    .toast-progress {
        height: 3px;
        width: 100%;
        animation: toast-shrink linear forwards;
        transform-origin: left;
        opacity: 0.8;
    }

    :global(.toast:hover .toast-progress),
    .page-hidden :global(.toast-progress) {
        animation-play-state: paused;
    }

    @keyframes toast-shrink {
        from {
            width: 100%;
        }
        to {
            width: 0%;
        }
    }
</style>
