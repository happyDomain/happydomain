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
        Alert,
        Button,
        Card,
        CardBody,
        CardHeader,
        Icon,
    } from "@sveltestrap/sveltestrap";

    import { deleteSessions } from '$lib/api/sessions';
    import { toasts } from '$lib/stores/toasts';

    let isDeleting = $state(false);

    async function handleDeleteAllSessions() {
        if (confirm('Are you sure you want to delete ALL sessions? This will log out all users!')) {
            isDeleting = true;
            try {
                await deleteSessions();
                toasts.addToast({
                    message: 'All sessions have been deleted successfully',
                    type: 'success',
                    timeout: 5000,
                });
            } catch (error) {
                toasts.addErrorToast({
                    message: 'Failed to delete sessions: ' + error,
                    timeout: 10000,
                });
            } finally {
                isDeleting = false;
            }
        }
    }
</script>

<Card>
    <CardHeader>
        <h5 class="mb-0">Delete All Sessions</h5>
    </CardHeader>
    <CardBody>
        <Alert color="warning" class="mb-3">
            <Icon name="exclamation-triangle" class="me-2"></Icon>
            <strong>Warning:</strong> This action will delete all active sessions and log out all users.
        </Alert>

        <Button
            color="danger"
            disabled={isDeleting}
            onclick={handleDeleteAllSessions}
        >
            <Icon name="trash" class="me-2"></Icon>
            {isDeleting ? 'Deleting...' : 'Delete All Sessions'}
        </Button>
    </CardBody>
</Card>
