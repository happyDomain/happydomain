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
      <b-spinner variant="primary" :label="$t('common.spinning')" class="mr-3" /> {{ $t('wait.wait') }}
    </div>

    <b-form v-else ref="form" class="mt-2" @submit.stop.prevent="goResend">
      <p>
        {{ $t('email.instruction.validate-address') }}
      </p>
      <p>
        {{ $t('email.instruction.new-confirmation') }}
      </p>
      <b-form-row>
        <label for="email-input" class="col-md-4 col-form-label text-truncate text-md-right font-weight-bold">{{ $t('email.address') }}</label>
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
          {{ $t('email.send-again') }}
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
      error: null
    }
  },

  computed: {
    isLoading () {
      return this.error === null
    }
  },

  mounted () {
    if (this.$route.query.u) {
      axios
        .post('/api/users/' + encodeURIComponent(btoa(this.$route.query.u.match(/\w{2}/g).map(function (a) {
          return String.fromCharCode(parseInt(a, 16))
        }).join(''))) + '/email', {
          key: this.$route.query.k
        })
        .then(
          (response) => {
            this.error = ''
            this.$root.$bvToast.toast(
              this.$t('account.ready-login'), {
                title: this.$t('email.instruction.validated'),
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/login')
          },
          (error) => {
            this.error = error.response.data.errmsg
          }
        )
    } else {
      this.error = ''
    }
  },

  methods: {
    goResend () {
      const valid = this.$refs.form.checkValidity()
      this.emailState = valid

      if (valid) {
        axios
          .patch('/api/users', {
            kind: 'validation',
            email: this.email
          })
          .then(
            (response) => {
              this.$root.$bvToast.toast(
                this.$t('email.instruction.check-inbox'), {
                  title: this.$t('email.sent'),
                  autoHideDelay: 5000,
                  variant: 'success',
                  toaster: 'b-toaster-content-right'
                }
              )
              this.$router.push('/email-validation')
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
