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
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { onMount } from "svelte";

    import { Container, Icon } from "@sveltestrap/sveltestrap";

    import { openReportModal } from "$lib/stores/report";
    import { t } from "$lib/translations";

    // When it happened, frozen: the details must not drift while the user
    // reads them.
    const occurredAt = new Date();

    let canGoBack = $state(false);

    onMount(() => {
        // An error in the root layout load leaves this page rendered on its
        // own: the splash has no other chance to be taken down.
        document.getElementById("app-splash")?.remove();

        canGoBack = window.history.length > 1;
    });

    const kind = $derived(
        page.status === 404
            ? "not-found"
            : page.status === 401 || page.status === 403
              ? "forbidden"
              : "unexpected",
    );

    // SvelteKit's own wording only repeats the status: it says nothing the
    // title doesn't already.
    const detail = $derived(
        page.error?.message && !["Not Found", "Internal Error"].includes(page.error.message)
            ? page.error.message
            : null,
    );

    function report() {
        // The route pattern, not the path: it locates the bug without
        // publishing the domain name the path may hold.
        const where = page.route.id ?? "an unknown route";
        openReportModal(`Error ${page.status} on ${where}` + (detail ? `: ${detail}` : ""));
    }
</script>

<div class="error-page">
    <Container class="py-4 py-lg-5">
        <div class="error-card">
            <div class="error-art">
                <img
                    src="/img/error-illustration.webp"
                    alt="A person and a cat sitting on a rock, looking at a city and a blank signpost"
                />
            </div>

            <div class="error-body">
                <p class="error-code" aria-hidden="true">{page.status}</p>
                <h1 class="h2 mb-3">{$t(`errors.page.${kind}.title`)}</h1>
                <p class="lead text-body-secondary">{$t(`errors.page.${kind}.content`)}</p>

                {#if detail}
                    <div class="error-detail">
                        <Icon name="info-circle" class="flex-shrink-0" />
                        <span>{detail}</span>
                    </div>
                {/if}

                <div class="d-flex flex-wrap gap-2 mt-4">
                    {#if kind === "unexpected"}
                        <button type="button" class="btn btn-primary" onclick={report}>
                            <Icon name="bug" class="me-1" />
                            {$t("report.report-this")}
                        </button>
                        <a href={resolve("/")} class="btn btn-outline-secondary">
                            <Icon name="house" class="me-1" />
                            {$t("errors.page.home")}
                        </a>
                    {:else}
                        <a href={resolve("/")} class="btn btn-primary">
                            <Icon name="house" class="me-1" />
                            {$t("errors.page.home")}
                        </a>
                    {/if}
                    {#if canGoBack}
                        <button
                            type="button"
                            class="btn btn-outline-secondary"
                            onclick={() => window.history.back()}
                        >
                            <Icon name="arrow-left" class="me-1" />
                            {$t("errors.page.back")}
                        </button>
                    {/if}
                    {#if kind !== "unexpected"}
                        <button
                            type="button"
                            class="btn btn-link text-decoration-none"
                            onclick={report}
                        >
                            <Icon name="bug" class="me-1" />
                            {$t("report.report-this")}
                        </button>
                    {/if}
                </div>

                <details class="error-tech mt-4">
                    <summary>{$t("errors.page.technical")}</summary>
                    <dl>
                        <dt>{$t("errors.page.status")}</dt>
                        <dd>{page.status}</dd>
                        <dt>{$t("errors.page.route")}</dt>
                        <dd>{page.route.id ?? "?"}</dd>
                        {#if page.error?.message}
                            <dt>{$t("errors.page.message")}</dt>
                            <dd>{page.error.message}</dd>
                        {/if}
                        <dt>{$t("errors.page.time")}</dt>
                        <dd>{occurredAt.toISOString()}</dd>
                    </dl>
                </details>
            </div>
        </div>
    </Container>
</div>

<style>
    .error-page {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
        justify-content: center;
        background: var(--hd-purple-50);
    }

    :global([data-bs-theme="dark"]) .error-page {
        background: color-mix(in srgb, var(--hd-purple-950) 50%, var(--bs-body-bg));
    }

    .error-card {
        display: grid;
        grid-template-columns: minmax(0, 2fr) minmax(0, 3fr);
        max-width: 60rem;
        margin: 0 auto;
        overflow: hidden;
        background: var(--hd-bg-default);
        border: 1px solid var(--hd-border-1);
        border-radius: 1rem;
    }

    .error-art img {
        display: block;
        width: 100%;
        height: 100%;
        object-fit: cover;
    }

    .error-body {
        display: flex;
        flex-direction: column;
        justify-content: center;
        padding: 2.5rem;
    }

    .error-code {
        margin: 0 0 0.25rem;
        font-family: var(--hd-font-title);
        font-size: clamp(3.5rem, 9vw, 6rem);
        font-weight: 800;
        line-height: 1;
        letter-spacing: -0.04em;
        align-self: flex-start;
        background: linear-gradient(120deg, var(--hd-accent), var(--hd-purple-500));
        -webkit-background-clip: text;
        background-clip: text;
        color: transparent;
    }

    .error-detail {
        display: flex;
        gap: 0.5rem;
        align-items: baseline;
        padding: 0.75rem 1rem;
        background: var(--hd-danger-bg);
        border: 1px solid var(--hd-danger-border);
        border-radius: 0.5rem;
        color: var(--hd-danger-text);
        overflow-wrap: anywhere;
    }

    .error-tech summary {
        color: var(--hd-fg-3);
        font-size: 0.875rem;
        cursor: pointer;
    }

    .error-tech dl {
        display: grid;
        grid-template-columns: max-content minmax(0, 1fr);
        gap: 0.25rem 1rem;
        margin: 0.75rem 0 0;
        padding: 0.75rem 1rem;
        background: var(--hd-bg-inset);
        border-radius: 0.5rem;
        font-size: 0.8125rem;
    }

    .error-tech dt {
        color: var(--hd-fg-3);
        font-weight: normal;
    }

    .error-tech dd {
        margin: 0;
        font-family: var(--hd-font-mono);
        overflow-wrap: anywhere;
    }

    @media (max-width: 767.98px) {
        .error-card {
            grid-template-columns: 1fr;
        }

        /* The portrait illustration would push everything below the fold:
           a band of it is enough. */
        .error-art {
            aspect-ratio: 16 / 9;
        }

        .error-art img {
            object-position: center 65%;
        }

        .error-body {
            padding: 1.5rem;
        }
    }
</style>
