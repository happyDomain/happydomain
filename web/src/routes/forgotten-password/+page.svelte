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
    import { Alert, Col, Container, Icon, Row } from "@sveltestrap/sveltestrap";

    import Logo from "$lib/components/Logo.svelte";
    import { appConfig } from "$lib/stores/config";
    import { t } from "$lib/translations";
    import ForgottenPasswordForm from "./ForgottenPasswordForm.svelte";
    import RecoverAccountForm from "./RecoverAccountForm.svelte";

    let { data } = $props();
</script>

<svelte:head>
    <title>{$t("password.forgotten")} - happyDomain</title>
</svelte:head>

<Container class="my-auto py-4">
    <Row class="justify-content-center align-items-stretch g-0">
        <Col md="5" lg="4" class="d-none d-md-block">
            <div class="recover-visual">
                <div class="text-center">
                    <Icon name="shield-lock" class="recover-icon" />
                </div>
            </div>
        </Col>
        <Col sm="10" md="7" lg="5">
            <div class="recover-card">
                <div class="text-center mb-4 d-md-none">
                    <Logo height="32" />
                </div>
                {#if $appConfig.no_mail}
                    <Alert color="warning" class="mb-0">
                        <h6 class="alert-heading fw-bold">{$t("password.recovery-unavailable.title")}</h6>
                        <p class="mb-0 small">
                            {$t("password.recovery-unavailable.description")}
                        </p>
                    </Alert>
                {:else if data.user && data.key}
                    <h5 class="fw-bold mb-1">{$t("password.redefine")}</h5>
                    <p class="text-body-secondary small mb-4">{$t("password.fill")}</p>
                    <RecoverAccountForm user={data.user} key={data.key} />
                {:else}
                    <h5 class="fw-bold mb-1">{$t("password.forgotten")}</h5>
                    <p class="text-body-secondary small mb-4">{$t("email.recover")}.</p>
                    <ForgottenPasswordForm />
                {/if}
                <div class="text-center mt-4 pt-3 border-top">
                    <a href="/login" class="text-body-secondary text-decoration-none small">
                        <Icon name="arrow-left" class="me-1" />
                        {$t("common.go-back")}
                    </a>
                </div>
            </div>
        </Col>
    </Row>
</Container>

<style>
    .recover-visual {
        height: 100%;
        border-radius: 1rem 0 0 1rem;
        overflow: hidden;
        background: linear-gradient(135deg, #f5f0f8 0%, #ede4f3 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 2rem;
    }

    .recover-visual :global(.recover-icon) {
        font-size: 5rem;
        color: #9332bb;
        opacity: 0.6;
    }

    .recover-card {
        background: #fff;
        border-radius: 0 1rem 1rem 0;
        padding: 2.5rem 2rem;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
        height: 100%;
        display: flex;
        flex-direction: column;
        justify-content: center;
    }

    @media (max-width: 767.98px) {
        .recover-card {
            border-radius: 1rem;
        }
    }
</style>
