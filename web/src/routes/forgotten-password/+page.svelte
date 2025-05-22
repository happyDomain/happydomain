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
    import { Alert, Col, Container, Row } from "@sveltestrap/sveltestrap";

    import { appConfig } from "$lib/stores/config";
    import { t } from "$lib/translations";
    import ForgottenPasswordForm from "./ForgottenPasswordForm.svelte";
    import RecoverAccountForm from "./RecoverAccountForm.svelte";

    let { data } = $props();
</script>

<Container class="my-3">
    {#if $appConfig.no_mail}
        <Row>
            <Col md={{ offset: 1, size: 10 }}  lg={{ offset: 2, size: 8 }} xl={{ offset: 3, size: 6 }}>
                <Alert color="warning">
                    <h4 class="alert-heading">{$t("password.recovery-unavailable.title")}</h4>
                    <p>
                        {$t("password.recovery-unavailable.description")}
                    </p>
                </Alert>
            </Col>
        </Row>
    {:else}
        {#if data.user && data.key}
            <RecoverAccountForm user={data.user} key={data.key} />
        {:else}
            <ForgottenPasswordForm />
        {/if}
    {/if}
</Container>
