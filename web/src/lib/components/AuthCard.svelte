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
    import type { Snippet } from "svelte";

    import { Container } from "@sveltestrap/sveltestrap";

    interface Props {
        src: string;
        alt: string;
        // Which part of the portrait illustration stays in the band shown
        // on narrow screens.
        mobilePosition?: string;
        children: Snippet;
    }

    let { src, alt, mobilePosition = "center", children }: Props = $props();
</script>

<div class="auth-page">
    <Container class="py-4 py-lg-5">
        <div class="auth-card">
            <div class="auth-art" style:--auth-art-position={mobilePosition}>
                <img {src} {alt} />
            </div>

            <div class="auth-body">
                {@render children()}
            </div>
        </div>
    </Container>
</div>

<style>
    .auth-page {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
        justify-content: center;
        background: var(--hd-bg-subtle);
    }

    .auth-card {
        display: grid;
        grid-template-columns: minmax(0, 2fr) minmax(0, 3fr);
        max-width: 60rem;
        margin: 0 auto;
        overflow: hidden;
        background: var(--hd-bg-canvas);
        border: 1px solid var(--hd-border-1);
        border-radius: 1rem;
    }

    .auth-art img {
        display: block;
        width: 100%;
        height: 100%;
        object-fit: cover;
    }

    .auth-body {
        display: flex;
        flex-direction: column;
        justify-content: center;
        padding: 2.5rem;
    }

    @media (max-width: 767.98px) {
        .auth-card {
            grid-template-columns: 1fr;
        }

        /* The portrait illustration would push the form below the fold:
           a band of it is enough. */
        .auth-art {
            max-height: 30vh;
        }

        .auth-art img {
            object-position: var(--auth-art-position);
        }

        .auth-body {
            padding: 1.5rem;
        }
    }
</style>
