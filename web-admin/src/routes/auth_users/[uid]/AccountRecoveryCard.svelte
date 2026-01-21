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
    import {
        Button,
        Card,
        CardBody,
        CardHeader,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import {
        postAuthByUidRecoverLink,
        postAuthByUidSendRecoverEmail,
    } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    interface Props {
        uid: string;
    }

    let {
        uid,
    }: Props = $props();

    let actionLoading = $state('');

    async function handleGenerateRecoveryLink() {
        actionLoading = 'recovery_link';
        try {
            const response = await postAuthByUidRecoverLink({ path: { uid } });
            if (response.data) {
                await navigator.clipboard.writeText(response.data);
            }
            toasts.addToast({
                message: 'Recovery link generated and copied to clipboard',
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: 'Failed to generate recovery link: ' + error,
                timeout: 10000,
            });
        } finally {
            actionLoading = '';
        }
    }

    async function handleSendRecoveryEmail() {
        actionLoading = 'send_recovery';
        try {
            await postAuthByUidSendRecoverEmail({ path: { uid } });
            toasts.addToast({
                message: 'Recovery email sent successfully',
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: 'Failed to send recovery email: ' + error,
                timeout: 10000,
            });
        } finally {
            actionLoading = '';
        }
    }
</script>

<Card class="mb-4">
    <CardHeader>
        <h5 class="mb-0">
            <Icon name="key"></Icon>
            Account Recovery
        </h5>
    </CardHeader>
    <CardBody class="d-flex flex-column gap-2">
        <Button
            color="warning"
            outline
            onclick={handleGenerateRecoveryLink}
            disabled={actionLoading !== ''}
        >
            {#if actionLoading === 'recovery_link'}
                <Spinner size="sm" class="me-2" />
            {:else}
                <Icon name="link-45deg" class="me-2"></Icon>
            {/if}
            Generate Recovery Link
        </Button>

        <Button
            color="warning"
            outline
            onclick={handleSendRecoveryEmail}
            disabled={actionLoading !== ''}
        >
            {#if actionLoading === 'send_recovery'}
                <Spinner size="sm" class="me-2" />
            {:else}
                <Icon name="envelope" class="me-2"></Icon>
            {/if}
            Send Recovery Email
        </Button>
    </CardBody>
</Card>
