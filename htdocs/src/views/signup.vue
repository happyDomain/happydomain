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
  <b-container style="margin-top: 10vh; margin-bottom: 10vh;">
    <b-row>
      <b-col sm="4" class="d-none d-sm-flex flex-column align-items-center">
        <img src="img/screenshot.png" alt="HappyDNS screenshoot" style="max-height: 100%; max-width: 100%;" class="mt-auto">
        <div class="mt-3 mb-auto text-justify">
          Join now our open source and free (as freedom) DNS platform, to manage your domains easily!
        </div>
      </b-col>
      <b-col sm="8">
        <b-card header-tag="div">
          <template v-slot:header>
            <h6 class="mb-0 font-weight-bold">
              {{ $t('account.signup.join-call') }}
            </h6>
          </template>
          <form ref="form" class="container mt-2" @submit.stop.prevent="goSignUp">
            <b-form-group
              :state="emailState"
              :label="$t('email.address')"
              label-for="email-input"
              :invalid-feedback="$t('errors.address-valid')"
            >
              <template v-slot:description>
                <i18n path="account.signup.address-why">
                  <template v-slot:identify>
                    <strong>{{ $t('account.signup.identify') }}</strong>
                  </template>
                  <template v-slot:security-operations>
                    <strong>{{ $t('account.signup.security-operations') }}</strong>
                  </template>
                </i18n>
              </template>
              <b-form-input
                id="email-input"
                ref="signupemail"
                v-model="signupForm.email"
                :state="emailState"
                required
                autofocus
                lazy
                type="email"
                placeholder="jPostel@isi.edu"
                autocomplete="username"
              />
            </b-form-group>
            <b-form-group
              :state="passwordState"
              :label="$t('common.password')"
              label-for="password-input"
              :invalid-feedback="$t('errors.password-weak')"
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
              :label="$t('password.confirmation')"
              label-for="passwordconfirm-input"
              :invalid-feedback="$t('errors.password-match')"
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
            <b-form-group>
              <b-form-checkbox
                v-model="signupForm.wantReceiveUpdate"
              >
                {{ $t('account.signup.receive-update') }}
              </b-form-checkbox>
            </b-form-group>
            <div class="d-flex justify-content-around">
              <b-button type="submit" variant="primary">
                {{ $t('account.signup.signup') }}
              </b-button>
              <b-button to="/login" variant="outline-dark">
                {{ $t('account.signup.already') }}
              </b-button>
            </div>
          </form>
        </b-card>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import axios from 'axios'
import PasswordChecks from '@/mixins/passwordChecks'

export default {

  mixins: [PasswordChecks],

  data: function () {
    return {
      signupForm: {
        email: '',
        password: '',
        passwordConfirm: ''
      }
    }
  },

  computed: {
    emailState () {
      if (this.signupForm.email.length === 0) {
        return null
      }
      return /.+@.+\..+/i.test(this.signupForm.email)
    }
  },

  methods: {
    goSignUp () {
      const valid = this.$refs.form.checkValidity()
      this.signupForm.emailState = valid ? 'valid' : 'invalid'
      this.signupForm.passwordState = valid ? 'valid' : 'invalid'

      if (valid) {
        axios
          .post('/api/users', {
            email: this.signupForm.email,
            password: this.signupForm.password
          })
          .then(
            (response) => {
              this.$root.$bvToast.toast(this.$t('email.instruction.check-inbox'), {
                title: this.$t('account.signup.success'),
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              })
              this.$router.push('/login')
            },
            (error) => {
              this.$bvToast.toast(
                error.response.data.errmsg, {
                  title: this.$t('errors.registration'),
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
