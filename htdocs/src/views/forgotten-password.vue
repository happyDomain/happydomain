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
  <b-container style="padding-top: 10vh; padding-bottom: 10vh;">
    <b-alert v-if="error !== null" variant="danger" :show="error.length > 0">
      {{ error }}
    </b-alert>

    <div v-if="isLoading" class="text-center">
      <b-spinner variant="primary" label="Spinning" class="mr-3" /> Please wait
    </div>

    <b-form v-else-if="user === ''" ref="formMail" @submit.stop.prevent="goSendLink">
      <p>
        In order to recover your account, we'll send you an e-mail containing a link that will allow you to redefine your password.
      </p>
      <b-form-row>
        <label for="email-input" class="col-md-4 col-form-label text-truncate text-md-right font-weight-bold">Email address</label>
        <b-col md="6">
          <b-form-input
            id="email-input"
            ref="signupemail"
            v-model="email"
            :state="emailState"
            required
            autofocus
            type="email"
            placeholder="jPostel@isi.edu"
            autocomplete="username"
          />
        </b-col>
      </b-form-row>
      <b-form-row class="mt-3">
        <b-button class="offset-sm-4 col-sm-4" type="submit" variant="primary">
          Send me an e-mail to recover my account
        </b-button>
      </b-form-row>
    </b-form>

    <b-form v-else ref="formRecover" @submit.stop.prevent="goRecover">
      <p>
        In order to recover your account, please fill the following form, with a fresh password.
      </p>
      <b-form-row>
        <label for="password-input" class="col-md-4 col-form-label text-truncate text-md-right font-weight-bold">New password</label>
        <b-col md="6">
          <b-form-input
            id="password-input"
            ref="recoverpassword"
            v-model="password"
            type="password"
            :state="passwordState"
            required
            placeholder="xXxXxXxXxX"
            autocomplete="new-password"
          />
        </b-col>
      </b-form-row>
      <b-form-row class="mt-2">
        <label for="passwordconfirm-input" class="col-md-4 col-form-label text-truncate text-md-right font-weight-bold">Password confirmation</label>
        <b-col md="6">
          <b-form-input
            id="passwordconfirm-input"
            ref="recoverpasswordconfirm"
            v-model="passwordConfirm"
            type="password"
            :state="passwordConfirmState"
            required
            placeholder="xXxXxXxXxX"
          />
        </b-col>
      </b-form-row>
      <b-form-row class="mt-3">
        <b-button class="offset-sm-4 col-sm-4" type="submit" variant="primary">
          Redefine my password
        </b-button>
      </b-form-row>
    </b-form>
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      email: '',
      emailState: null,
      error: null,
      password: '',
      passwordConfirm: '',
      user: null
    }
  },

  computed: {
    isLoading () {
      return this.error === null || this.user === null
    },
    passwordState () {
      if (this.password.length === 0) {
        return null
      }
      return this.password.length > 15 || (
        /[A-Z]/.test(this.password) && /[a-z]/.test(this.password) && /[0-9]/.test(this.password) && (/\W/.test(this.password) || this.password.length >= 8))
    },
    passwordConfirmState () {
      if (this.passwordConfirm.length === 0) {
        return null
      }
      return this.password === this.passwordConfirm
    }
  },

  mounted () {
    if (this.$route.query.u) {
      axios
        .post('/api/users/' + encodeURIComponent(this.$route.query.u) + '/recovery', {
          key: this.$route.query.k
        })
        .then(
          (response) => {
            this.error = ''
            this.user = this.$route.query.u
          },
          (error) => {
            this.error = error.response.data.errmsg
            this.user = ''
          }
        )
    } else {
      this.error = ''
      this.user = ''
    }
  },

  methods: {
    goSendLink () {
      const valid = this.$refs.formMail.checkValidity()
      this.emailState = valid

      if (valid) {
        axios
          .patch('/api/users', {
            kind: 'recovery',
            email: this.email
          })
          .then(
            (response) => {
              this.$root.$bvToast.toast(
                'Please check your inbox in order to recover your account.', {
                  title: 'Password recovery email send!',
                  autoHideDelay: 5000,
                  variant: 'success',
                  toaster: 'b-toaster-content-right'
                }
              )
              this.$router.push('/login')
            },
            (error) => {
              this.$bvToast.toast(
                error.response.data.errmsg, {
                  title: 'Password recovery problem',
                  autoHideDelay: 5000,
                  variant: 'danger',
                  toaster: 'b-toaster-content-right'
                }
              )
            }
          )
      }
    },
    goRecover () {
      const valid = this.$refs.formRecover.checkValidity()

      if (valid && this.user) {
        axios
          .post('/api/users/' + encodeURIComponent(this.user) + '/recovery', {
            key: this.$route.query.k,
            password: this.password
          })
          .then(
            (response) => {
              this.$root.$bvToast.toast(
                'You can now login with your new password.', {
                  title: 'Password redefined successfully!',
                  autoHideDelay: 5000,
                  variant: 'success',
                  toaster: 'b-toaster-content-right'
                }
              )
              this.$router.push('/login')
            },
            (error) => {
              this.$bvToast.toast(
                error.response.data.errmsg, {
                  title: 'Password recovery problem',
                  autoHideDelay: 5000,
                  variant: 'danger',
                  toaster: 'b-toaster-content-right'
                }
              )
            }
          )
      }
    }
  }
}
</script>
