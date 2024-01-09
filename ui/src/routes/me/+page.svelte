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
 import {
     Card,
     CardBody,
     Container,
     Col,
     Row,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import ChangePasswordForm from '$lib/components/ChangePasswordForm.svelte';
 import DeleteAccountCard from '$lib/components/DeleteAccountCard.svelte';
 import UserSettingsForm from '$lib/components/UserSettingsForm.svelte';
 import { t } from '$lib/translations';
 import { userSession } from '$lib/stores/usersession';
</script>

<Container class="my-4">
    <h2 id="settings">
        {$t('settings.title')}
    </h2>
    {#if !$userSession}
        <div class="d-flex justify-content-center">
            <Spinner color="primary" />
        </div>
    {:else}
        <Row>
            {#if $userSession.settings}
                <Card class="offset-md-2 col-8">
                    <CardBody>
                        <UserSettingsForm settings={$userSession.settings} />
                    </CardBody>
                </Card>
            {/if}
        </Row>
        {#if $userSession.email !== '_no_auth'}
            <h2 id="password-change">
                {$t('password.change')}
            </h2>
            <Row>
                <Col md={{size: 8, offset: 2}}>
                <Card>
                    <CardBody>
                        <ChangePasswordForm />
                    </CardBody>
                </Card>
            </Col>
            </Row>
            <hr>
            <h2 id="delete-account">
                {$t('account.delete.delete')}
            </h2>
            <Row>
                <Col md={{size: 8, offset: 2}}>
                <DeleteAccountCard />
                </Col>
            </Row>
        {:else}
            <div class="m-5 alert alert-secondary">
                {$t('errors.account-no-auth')}
            </div>
        {/if}
    {/if}
</Container>
