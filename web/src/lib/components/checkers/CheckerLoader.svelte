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
    import { Icon } from "@sveltestrap/sveltestrap";

    interface Props {
        label?: string;
        icon?: string;
    }

    let { label = "", icon = "search" }: Props = $props();
</script>

<div class="happy-loader" role="status" aria-live="polite">
    <div class="happy-loader-radar">
        <span class="ring ring-1"></span>
        <span class="ring ring-2"></span>
        <span class="ring ring-3"></span>
        <span class="sweep"></span>
        <span class="blip blip-a"></span>
        <span class="blip blip-b"></span>
        <span class="blip blip-c"></span>
        <span class="core">
            <Icon name={icon} />
        </span>
    </div>
    {#if label}
        <p class="happy-loader-title mt-4 mb-0">{label}</p>
    {/if}
</div>

<style>
    .happy-loader {
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
        padding: 2rem 1rem;
        max-width: 28rem;
    }

    .happy-loader-radar {
        position: relative;
        width: 180px;
        height: 180px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .happy-loader-radar .ring {
        position: absolute;
        border-radius: 50%;
        border: 1.5px solid color-mix(in srgb, var(--hd-accent) 45%, transparent);
        opacity: 0;
        animation: ring-pulse 2.8s ease-out infinite;
    }
    .happy-loader-radar .ring-1 { width: 60px; height: 60px; animation-delay: 0s;   }
    .happy-loader-radar .ring-2 { width: 60px; height: 60px; animation-delay: 0.9s; }
    .happy-loader-radar .ring-3 { width: 60px; height: 60px; animation-delay: 1.8s; }

    @keyframes ring-pulse {
        0%   { width: 60px;  height: 60px;  opacity: 0.8; }
        80%  { width: 170px; height: 170px; opacity: 0;   }
        100% { width: 170px; height: 170px; opacity: 0;   }
    }

    .happy-loader-radar .sweep {
        position: absolute;
        inset: 0;
        border-radius: 50%;
        background: conic-gradient(
            from 0deg,
            color-mix(in srgb, var(--hd-accent) 55%, transparent) 0deg,
            color-mix(in srgb, var(--hd-accent) 18%, transparent) 40deg,
            transparent 90deg,
            transparent 360deg
        );
        mask: radial-gradient(circle, transparent 18px, #000 22px, #000 88px, transparent 90px);
        -webkit-mask: radial-gradient(circle, transparent 18px, #000 22px, #000 88px, transparent 90px);
        animation: sweep-rot 3.2s linear infinite;
    }

    @keyframes sweep-rot {
        to { transform: rotate(-360deg); }
    }

    .happy-loader-radar .blip {
        position: absolute;
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: var(--hd-accent);
        box-shadow: 0 0 8px color-mix(in srgb, var(--hd-accent) 70%, transparent);
        opacity: 0;
        animation: blip-flash 3.2s ease-in-out infinite;
    }
    .happy-loader-radar .blip-a { top: 28%; left: 70%; animation-delay: 0.6s; }
    .happy-loader-radar .blip-b { top: 66%; left: 30%; animation-delay: 1.7s; }
    .happy-loader-radar .blip-c { top: 50%; left: 78%; animation-delay: 2.4s; }

    @keyframes blip-flash {
        0%, 100% { opacity: 0; transform: scale(0.6); }
        20%      { opacity: 1; transform: scale(1);   }
        60%      { opacity: 0; transform: scale(0.6); }
    }

    .happy-loader-radar .core {
        position: relative;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 56px;
        height: 56px;
        border-radius: 50%;
        background: var(--hd-bg-canvas);
        color: var(--hd-accent);
        font-size: 1.4rem;
        box-shadow:
            0 0 0 1px var(--hd-accent-border),
            0 6px 18px color-mix(in srgb, var(--hd-accent) 25%, transparent);
        animation: core-bob 2.4s ease-in-out infinite;
    }

    @keyframes core-bob {
        0%, 100% { transform: translateY(0)    scale(1);    }
        50%      { transform: translateY(-3px) scale(1.04); }
    }

    .happy-loader-title {
        font-weight: 600;
        font-size: 1.05rem;
        color: var(--hd-fg-1);
    }

    @media (prefers-reduced-motion: reduce) {
        .happy-loader-radar .ring,
        .happy-loader-radar .sweep,
        .happy-loader-radar .blip,
        .happy-loader-radar .core {
            animation: none;
        }
        .happy-loader-radar .ring { opacity: 0.4; width: 120px; height: 120px; }
    }
</style>
