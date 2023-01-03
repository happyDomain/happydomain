<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Button,
     Card,
     CardBody,
     Input,
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
     Spinner,
 } from 'sveltestrap';

 import { deleteUserAccount } from '$lib/api/user';
 import { t } from '$lib/translations';
 import { userSession } from '$lib/stores/usersession';
 import { toasts } from '$lib/stores/toasts';

 let deleteAccountModalOpen = false;
 let password = "";
 let formSent = false;

 $: if (deleteAccountModalOpen) password = "";

 function deleteMyAccount() {
     if ($userSession == null) {
         return
     }

     formSent = true;
     deleteUserAccount($userSession, password).then(
         () => {
             formSent = false;
             deleteAccountModalOpen = false;
             toasts.addToast({
                 title: $t('account.delete.deleted'),
                 message: $t('account.delete.success'),
                 type: 'primary',
             });
             goto('/login');
         },
         (err) => {
             formSent = false;
             toasts.addErrorToast({
                 title: $t('errors.account-delete'),
                 message: err,
                 timeout: 5000,
             });
         }
     );
 }
</script>

<Card {...$$restProps}>
    <CardBody>
        <p>
            {$t('account.delete.confirm')}
        </p>
        <Button
            type="button"
            color="danger"
            disabled={formSent}
            on:click={() => deleteAccountModalOpen = true}
        >
            {#if formSent}
                <Spinner size="sm" class="me-2" />
            {/if}
            {$t('account.delete.delete')}
        </Button>
        <p class="mt-2 text-muted" style="line-height: 1.1">
            <small>
                {$t('account.delete.consequence')}
            </small>
        </p>
    </CardBody>
</Card>

<Modal
    isOpen={deleteAccountModalOpen}
    toggle={() => deleteAccountModalOpen = !deleteAccountModalOpen}
>
    <ModalHeader
        toggle={() => deleteAccountModalOpen = !deleteAccountModalOpen}
    >
        {$t('account.delete.delete')}
    </ModalHeader>
    <ModalBody>
        <p>
            {$t('account.delete.confirm-twice')}
        </p>
        <div>
            <label for="currentPassword-forDeletion">
                {$t('account.delete.confirm-password')}
            </label>
            <Input
                id="currentPassword-forDeletion"
                class="border-danger"
                autocomplete="off"
                autofocus
                required
                placeholder="xXxXxXxXxX"
                type="password"
                bind:value={password}
            />
        </div>
        <p class="text-muted" style="line-height: 1.1">
            <small>
                {$t('account.delete.remain-data')}
            </small>
        </p>
    </ModalBody>
    <ModalFooter>
        <Button
            color="danger"
            on:click={deleteMyAccount}
        >
            {$t('account.delete.delete')}
        </Button>
        <Button
            color="secondary"
            on:click={() => deleteAccountModalOpen = !deleteAccountModalOpen}
        >
            {$t('common.cancel')}
        </Button>
    </ModalFooter>
</Modal>
