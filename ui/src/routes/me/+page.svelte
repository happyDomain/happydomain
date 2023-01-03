<script lang="ts">
 import {
     Card,
     CardBody,
     Container,
     Col,
     Row,
     Spinner,
 } from 'sveltestrap';

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
