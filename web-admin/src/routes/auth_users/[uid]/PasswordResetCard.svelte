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

    import { postAuthByUidResetPassword } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    interface Props {
        uid: string;
    }

    let {
        uid,
    }: Props = $props();

    let actionLoading = $state(false);

    async function handleResetPassword() {
        if (!confirm('Are you sure you want to reset this user\'s password? A random password will be generated.')) return;

        actionLoading = true;
        try {
            const response = await postAuthByUidResetPassword({
                path: { uid },
                body: { password: '' }
            });
            const password = response.data?.password || '';
            await navigator.clipboard.writeText(password);
            toasts.addToast({
                message: 'Password reset successfully and copied to clipboard',
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: 'Failed to reset password: ' + error,
                timeout: 10000,
            });
        } finally {
            actionLoading = false;
        }
    }
</script>

<Card class="mb-4">
    <CardHeader>
        <h5 class="mb-0">
            <Icon name="shield-lock"></Icon>
            Password Reset
        </h5>
    </CardHeader>
    <CardBody class="d-flex flex-column gap-2">
        <Button
            color="danger"
            outline
            onclick={handleResetPassword}
            disabled={actionLoading}
        >
            {#if actionLoading}
                <Spinner size="sm" class="me-2" />
            {:else}
                <Icon name="arrow-clockwise" class="me-2"></Icon>
            {/if}
            Reset Password (Generate Random)
        </Button>
    </CardBody>
</Card>
