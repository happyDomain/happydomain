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
    <h2 id="settings">
      {{ $t('settings.title') }}
    </h2>
    <b-row>
      <b-card v-if="settings" class="offset-md-2 col-8">
        <b-form @submit.stop.prevent="saveSettings">
          <b-form-group
            :label="$t('settings.language')"
            label-for="language-select"
          >
            <b-form-select id="language-select" v-model="settings.language" :options="languages" />
          </b-form-group>
          <b-form-group
            :label="$t('settings.fieldhint.title')"
            label-for="fieldhint-select"
          >
            <b-form-select
              id="fieldhint-select"
              v-model="settings.fieldhint"
              :options="[{value: 0, text: $t('settings.fieldhint.hide')}, {value: 1, text: $t('settings.fieldhint.tooltip')}, {value: 2, text: $t('settings.fieldhint.focused')}, {value: 3, text: $t('settings.fieldhint.always')}]"
            />
          </b-form-group>
          <b-form-group
            :label="$t('settings.zoneview.title')"
          >
            <b-button-group class="w-100">
              <b-button :variant="!settings.zoneview ? 'secondary' : 'outline-secondary'" @click="setSetting('zoneview', 0)">
                <b-icon icon="grid-fill" aria-hidden="true" /><br>
                {{ $t('settings.zoneview.grid') }}
              </b-button>
              <b-button :variant="settings.zoneview === 1 ? 'secondary' : 'outline-secondary'" @click="setSetting('zoneview', 1)">
                <b-icon icon="list-ul" aria-hidden="true" /><br>
                {{ $t('settings.zoneview.list') }}
              </b-button>
              <b-button :variant="settings.zoneview === 2 ? 'secondary' : 'outline-secondary'" @click="setSetting('zoneview', 2)">
                <b-icon icon="menu-button-wide-fill" aria-hidden="true" /><br>
                {{ $t('settings.zoneview.records') }}
              </b-button>
            </b-button-group>
          </b-form-group>
          <div class="d-flex justify-content-around">
            <b-button type="submit" variant="primary">
              {{ $t('settings.save') }}
            </b-button>
          </div>
        </b-form>
      </b-card>
    </b-row>
    <div v-if="loggedUser && loggedUser.email !== '_no_auth'">
      <h2 id="password-change">
        {{ $t('password.change') }}
      </h2>
      <b-row>
        <b-card class="offset-md-2 col-8">
          <b-form @submit.stop.prevent="sendChPassword">
            <b-form-group
              :label="$t('password.enter')"
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
              :label="$t('password.enter-new')"
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
              :label="$t('password.confirm-new')"
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
            <div class="d-flex justify-content-around">
              <b-button type="submit" variant="primary">
                {{ $t('password.change') }}
              </b-button>
            </div>
          </b-form>
        </b-card>
      </b-row>
      <hr>
      <h2 id="delete-account">
        {{ $t('account.delete.delete') }}
      </h2>
      <b-row>
        <b-card class="offset-md-2 col-8">
          <p>
            {{ $t('account.delete.confirm') }}
          </p>
          <b-button type="button" variant="danger" @click="askAccountDeletion">
            {{ $t('account.delete.delete') }}
          </b-button>
          <p class="mt-2 text-muted" style="line-height: 1.1">
            <small>
              {{ $t('account.delete.consequence') }}
            </small>
          </p>
        </b-card>
      </b-row>
    </div>
    <div v-else class="m-5 alert alert-secondary">
      {{ $t('errors.account-no-auth') }}
    </div>
    <b-modal id="delete-account-modal" :title="$t('account.delete.delete')" ok-variant="danger" :ok-title="$t('account.delete.delete')" cancel-variant="primary" @ok="deleteMyAccount">
      <p>
        {{ $t('account.delete.confirm-twice') }}
      </p>
      <b-form-group
        :label="$t('account.delete.confirm-password')"
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
          {{ $t('account.delete.remain-data') }}
        </small>
      </p>
    </b-modal>
  </b-container>
</template>

<script>
import axios from 'axios'
import Vue from 'vue'
import PasswordChecks from '@/mixins/passwordChecks'
import Languages from '@/mixins/languages'

export default {
  mixins: [PasswordChecks, Languages],

  data () {
    return {
      deletePassword: '',
      loggedUser: null,
      settings: null,
      signupForm: {
        current: '',
        password: '',
        passwordConfirm: ''
      }
    }
  },

  computed: {
    isLoading () {
      return this.loggedUser != null || this.settings != null
    }
  },

  created () {
    axios.get('/api/auth')
      .then(
        (response) => {
          this.loggedUser = response.data
          axios.get('/api/users/' + encodeURIComponent(this.loggedUser.id.toString(16)) + '/settings')
            .then(
              (response) => {
                this.settings = response.data
              })
        })
  },

  methods: {
    askAccountDeletion () {
      this.deletePassword = ''
      this.$bvModal.show('delete-account-modal')
    },

    deleteMyAccount () {
      axios
        .post('/api/users/' + encodeURIComponent(this.loggedUser.id.toString(16)) + '/delete', { current: this.deletePassword })
        .then(
          response => {
            this.$root.$bvToast.toast(
              this.$t('account.delete.success'), {
                title: this.$t('account.delete.deleted'),
                variant: 'primary',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/login')
          },
          error => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.account-delete'),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    setSetting (setting, value) {
      Vue.set(this.settings, setting, value)
    },

    saveSettings () {
      axios
        .post('/api/users/' + encodeURIComponent(this.loggedUser.id.toString(16)) + '/settings', this.settings)
        .then(
          response => {
            this.$store.dispatch('user/updateSettings')
            this.settings = response.data
            this.$root.$bvToast.toast(this.$t('settings.success'), {
              title: this.$t('settings.success-change'),
              autoHideDelay: 5000,
              variant: 'success',
              toaster: 'b-toaster-content-right'
            })
          },
          error => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.settings-change'),
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
            this.$root.$bvToast.toast(this.$t('password.success-change'), {
              title: this.$t('password.changed'),
              autoHideDelay: 5000,
              variant: 'success',
              toaster: 'b-toaster-content-right'
            })
            this.$router.push('/')
          },
          error => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.password-change'),
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
