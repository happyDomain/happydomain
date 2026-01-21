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
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import type { HappydnsUserAuth } from '$lib/api-admin';
    import {
        putAuthByUid,
        postAuthByUidValidationLink,
        postAuthByUidSendValidationEmail,
        postAuthByUidValidateEmail,
    } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    interface Props {
        authUser: HappydnsUserAuth;
        uid: string;
        onRefresh: () => void;
    }

    let {
        authUser,
        uid,
        onRefresh,
    }: Props = $props();

    let actionLoading = $state('');

    async function handleGenerateValidationLink() {
        actionLoading = 'validation_link';
        try {
            const response = await postAuthByUidValidationLink({ path: { uid } });
            if (response.data) {
                await navigator.clipboard.writeText(response.data);
            }
            toasts.addToast({
                message: 'Validation link generated and copied to clipboard',
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: 'Failed to generate validation link: ' + error,
                timeout: 10000,
            });
        } finally {
            actionLoading = '';
        }
    }

    async function handleSendValidationEmail() {
        actionLoading = 'send_validation';
        try {
            await postAuthByUidSendValidationEmail({ path: { uid } });
            toasts.addToast({
                message: 'Validation email sent successfully',
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: 'Failed to send validation email: ' + error,
                timeout: 10000,
            });
        } finally {
            actionLoading = '';
        }
    }

    async function handleValidateEmail() {
        if (!confirm('Are you sure you want to mark this email as verified?')) return;

        actionLoading = 'validate_email';
        try {
            await postAuthByUidValidateEmail({ path: { uid } });
            onRefresh();
            toasts.addToast({
                message: 'Email marked as verified successfully',
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: 'Failed to validate email: ' + error,
                timeout: 10000,
            });
        } finally {
            actionLoading = '';
        }
    }

    async function handleUnverifyEmail() {
        if (!confirm('Are you sure you want to mark this email as unverified?')) return;

        actionLoading = 'validate_email';
        try {
            await putAuthByUid({
                path: { uid },
                body: {
                    email: authUser.email,
                    emailVerification: undefined
                }
            });
            onRefresh();
            toasts.addToast({
                message: 'Email marked as unverified successfully',
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: 'Failed to unverify email: ' + error,
                timeout: 10000,
            });
        } finally {
            actionLoading = '';
        }
    }
</script>

<Card class="mb-4">
    <CardHeader>
        <h5 class="mb-0 d-flex align-items-center gap-2">
            <Icon name={authUser.emailVerification === null ? "envelope-exclamation" : "envelope-check"}></Icon>
            Email Verification
            {#if authUser.emailVerification !== null}
                <Badge color="success">Verified</Badge>
            {:else}
                <Badge color="danger">Not Verified</Badge>
            {/if}
        </h5>
    </CardHeader>
    <CardBody class="d-flex flex-column gap-2">
        {#if authUser.emailVerification !== null}
            <Button
                color="warning"
                outline
                onclick={handleUnverifyEmail}
                disabled={actionLoading !== ''}
            >
                {#if actionLoading === 'validate_email'}
                    <Spinner size="sm" class="me-2" />
                {:else}
                    <Icon name="x-circle" class="me-2"></Icon>
                {/if}
                Mark Email as Unverified
            </Button>
        {:else}
            <Button
                color="success"
                outline
                onclick={handleValidateEmail}
                disabled={actionLoading !== ''}
            >
                {#if actionLoading === 'validate_email'}
                    <Spinner size="sm" class="me-2" />
                {:else}
                    <Icon name="check-circle" class="me-2"></Icon>
                {/if}
                Mark Email as Verified
            </Button>
        {/if}

        <Button
            color="primary"
            outline
            onclick={handleGenerateValidationLink}
            disabled={actionLoading !== ''}
        >
            {#if actionLoading === 'validation_link'}
                <Spinner size="sm" class="me-2" />
            {:else}
                <Icon name="link-45deg" class="me-2"></Icon>
            {/if}
            Generate Validation Link
        </Button>

        <Button
            color="primary"
            outline
            onclick={handleSendValidationEmail}
            disabled={actionLoading !== ''}
        >
            {#if actionLoading === 'send_validation'}
                <Spinner size="sm" class="me-2" />
            {:else}
                <Icon name="envelope" class="me-2"></Icon>
            {/if}
            Send Validation Email
        </Button>
    </CardBody>
</Card>
