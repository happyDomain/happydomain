<!--
    Copyright or © or Copr. happyDNS (2020)

    contact@happydns.org

    This software is a computer program whose purpose is to provide a modern
    interface to interact with DNS systems.

    This software is governed by the CeCILL license under French law and abiding
    by the rules of distribution of free software.  You can use, modify and/or
    redistribute the software under the terms of the CeCILL license as
    circulated by CEA, CNRS and INRIA at the following URL
    "http://www.cecill.info".

    As a counterpart to the access to the source code and rights to copy, modify
    and redistribute granted by the license, users are provided only with a
    limited warranty and the software's author, the holder of the economic
    rights, and the successive licensors have only limited liability.

    In this respect, the user's attention is drawn to the risks associated with
    loading, using, modifying and/or developing or reproducing the software by
    the user in light of its specific status of free software, that may mean
    that it is complicated to manipulate, and that also therefore means that it
    is reserved for developers and experienced professionals having in-depth
    computer knowledge. Users are therefore encouraged to load and test the
    software's suitability as regards their requirements in conditions enabling
    the security of their systems and/or data to be ensured and, more generally,
    to use and operate it in the same conditions as regards security.

    The fact that you are presently reading this means that you have had
    knowledge of the CeCILL license and that you accept its terms.
  -->

<template>
  <b-container class="my-4">
    <h2 id="password-change">
      Change my password
    </h2>
    <b-row>
      <b-card class="offset-md-2 col-8">
        <b-form @submit.stop.prevent="sendChPassword">
          <b-form-group
            label="Enter your current password"
            label-for="currentPassword-input"
          >
            <b-form-input
              id="currentPassword-input"
              v-model="signupForm.current"
              type="password"
              required
              placeholder="xXxXxXxXxX"
              autocomplete="current-password"
            />
          </b-form-group>
          <b-form-group
            :state="passwordState"
            label="Enter your new password"
            label-for="password-input"
            invalid-feedback="Password has to be strong enough: at least 8 characters with numbers, low case characters and high case characters."
          >
            <b-form-input
              id="password-input"
              ref="signuppassword"
              v-model="signupForm.password"
              type="password"
              :state="passwordState"
              required
              placeholder="xXxXxXxXxX"
              autocomplete="new-password"
            />
          </b-form-group>
          <b-form-group
            :state="passwordConfirmState"
            label="Confirm your new password"
            label-for="passwordconfirm-input"
            invalid-feedback="Password and its confirmation doesn't match."
          >
            <b-form-input
              id="passwordconfirm-input"
              ref="signuppasswordconfirm"
              v-model="signupForm.passwordConfirm"
              type="password"
              :state="passwordConfirmState"
              required
              placeholder="xXxXxXxXxX"
            />
          </b-form-group>
          <div class="d-flex justify-content-around">
            <b-button type="submit" variant="primary">
              Change password
            </b-button>
          </div>
        </b-form>
      </b-card>
    </b-row>
    <hr>
    <h2 id="delete-account">
      Delete my account
    </h2>
    <b-row>
      <b-card class="offset-md-2 col-8">
        <p>
          If you want to delete your account and all data associated with it, press the button below:
        </p>
        <b-button type="button" variant="danger" @click="askAccountDeletion">
          Delete my account
        </b-button>
        <p class="mt-2 text-muted" style="line-height: 1.1">
          <small>
            Your domains owned on others platforms will not be affected by the deletion, they'll continue to respond with the current dataset.
          </small>
        </p>
      </b-card>
    </b-row>
    <b-modal id="delete-account-modal" title="Delete Your Account" ok-variant="danger" ok-title="Delete my account" cancel-variant="primary" @ok="deleteMyAccount">
      <p>
        By confirming the deletion, you'll permanently and irrevocably delete your account from our database and will loose easy access to our easy management interface for your domains.
      </p>
      <b-form-group
        label="To ensure this is really you, please enter your password:"
        label-for="currentPassword-forDeletion"
      >
        <b-form-input
          id="currentPassword-forDeletion"
          v-model="deletePassword"
          autocomplete="off"
          autofocus
          required
          placeholder="xXxXxXxXxX"
          style="border-color:#c92052"
          type="password"
        />
      </b-form-group>
      <p class="text-muted" style="line-height: 1.1">
        <small>
          For technical reason, your account will be deleted right after your validation, but some data from your account will persist until the next database clean up.
        </small>
      </p>
    </b-modal>
  </b-container>
</template>

<script>
import axios from 'axios'
import PasswordChecks from '@/mixins/passwordChecks'

export default {
  mixins: [PasswordChecks],

  data () {
    return {
      deletePassword: '',
      loggedUser: null,
      signupForm: {
        current: '',
        password: '',
        passwordConfirm: ''
      }
    }
  },

  computed: {
    isLoading () {
      return this.loggedUser != null
    }
  },

  created () {
    axios.get('/api/auth')
      .then(
        (response) => {
          this.loggedUser = response.data
        })
  },

  methods: {
    askAccountDeletion () {
      this.deletePassword = ''
      this.$bvModal.show('delete-account-modal')
    },

    deleteMyAccount () {
      axios
        .post('/api/users/' + encodeURIComponent(this.loggedUser.id.toString(16)) + '/delete', { password: this.deletePassword })
        .then(
          response => {
            this.$root.$bvToast.toast(
              'Your account have been successfully deleted. We hope to see you back soon.', {
                title: 'Account Deleted',
                variant: 'primary',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/login')
          },
          error => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when trying to delete your account',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    sendChPassword () {
      axios
        .post('/api/users/' + encodeURIComponent(this.loggedUser.id.toString(16)) + '/new_password', this.signupForm)
        .then(
          response => {
            this.$root.$bvToast.toast(
              'Your account\'s password has been changed with success.', {
                title: 'Password Successfully Changed',
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/')
          },
          error => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'Unable to change your password account',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    }
  }
}
</script>
