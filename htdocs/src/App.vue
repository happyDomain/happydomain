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
  <div id="app">
    <b-navbar :class="loggedUser?'p-0':''">
      <b-container>
        <b-navbar-brand class="navbar-brand" to="/">
          <h-logo height="25" />
        </b-navbar-brand>
        <button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#adminMenu" aria-controls="adminMenu" aria-expanded="false" aria-label="Toggle navigation">
          <span class="navbar-toggler-icon" />
        </button>

        <b-navbar-toggle target="nav-collapse" />

        <b-navbar-nav class="ml-auto">
          <b-nav-item-dropdown v-if="loggedUser" right>
            <template slot="button-content">
              <b-button size="sm" variant="dark">
                <b-icon icon="person" aria-hidden="true" /> {{ loggedUser.email }}
              </b-button>
            </template>
            <b-dropdown-item to="/domains/">
              {{ $t('menu.my-domains') }}
            </b-dropdown-item>
            <b-dropdown-item to="/sources/">
              {{ $t('menu.my-sources') }}
            </b-dropdown-item>
            <b-dropdown-divider />
            <b-dropdown-item to="/tools/client">
              {{ $t('menu.dns-client') }}
            </b-dropdown-item>
            <b-dropdown-divider />
            <b-dropdown-item to="/me">
              {{ $t('menu.my-account') }}
            </b-dropdown-item>
            <b-dropdown-divider />
            <b-dropdown-item @click="logout()">
              {{ $t('menu.logout') }}
            </b-dropdown-item>
          </b-nav-item-dropdown>
          <b-button v-if="!loggedUser" variant="outline-dark" to="/join">
            <b-icon icon="person-plus-fill" aria-hidden="true" /> {{ $t('menu.signup') }}
          </b-button>
          <b-button v-if="!loggedUser" variant="primary" class="ml-2" to="/login">
            <b-icon icon="person-check" aria-hidden="true" /> {{ $t('menu.signin') }}
          </b-button>
        </b-navbar-nav>
      </b-container>
    </b-navbar>

    <router-view :logged-user="loggedUser" style="min-height: 80vh" />

    <b-toaster name="b-toaster-content-right" style="position: fixed; top: 70px; right: 0; z-index: 1042; min-width: 30vw;" />

    <footer class="pt-2 pb-2 bg-dark text-light">
      <b-container>
        <b-row>
          <b-col md="12" lg="6">
            &copy;
            <h-logo color="#fff" height="17" />
            2019-2020 All rights reserved
          </b-col>
          <b-col md="6" lg="3" />
          <b-col md="6" lg="3" />
        </b-row>
      </b-container>
    </footer>
  </div>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      alreadyShownUpdate: false,
      loggedUser: undefined
    }
  },

  mounted () {
    if (sessionStorage.loggedUser) {
      this.loggedUser = JSON.parse(sessionStorage.loggedUser)
    }
    this.updateSession()
    this.$on('login', this.login)

    setInterval(function (vm) { vm.checkForUpdate() }, 360000, this)
    setTimeout(function (vm) { vm.checkForUpdate() }, 42000, this)
  },

  methods: {
    logout () {
      axios
        .post('/api/auth/logout')
        .then(
          (response) => {
            delete sessionStorage.loggedUser
            this.loggedUser = null
            this.updateSession()
            this.$router.push('/')
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.logout'),
                autoHideDelay: 5000,
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    },

    updateSession () {
      axios.get('/api/auth')
        .then(
          (response) => {
            sessionStorage.loggedUser = JSON.stringify(response.data)
            this.loggedUser = response.data
          },
          (error) => {
            this.loggedUser = null
            if (sessionStorage.loggedUser) {
              delete sessionStorage.loggedUser
              this.$root.$bvToast.toast(
                this.$t('errors.session.content', { err: error.response.data.errmsg }), {
                  title: this.$t('errors.session.title'),
                  autoHideDelay: 5000,
                  variant: 'danger',
                  toaster: 'b-toaster-content-right'
                }
              )
              this.$router.replace('/login')
            }
          }
        )
    },

    checkForUpdate (timeout) {
      if (sessionStorage.getItem('happyUpdate') && !this.alreadyShownUpdate) {
        this.alreadyShownUpdate = true
        this.$bvToast.toast(
          this.$t('upgrade.content'), {
            title: this.$t('upgrade.title'),
            variant: 'primary',
            href: window.location.pathname,
            noAutoHide: true,
            toaster: 'b-toaster-content-right'
          }
        )
      }
    },

    login (email, password) {
      axios
        .post('/api/auth', {
          email: email,
          password: password
        })
        .then(
          (response) => {
            sessionStorage.loggedUser = JSON.stringify(response.data)
            this.loggedUser = response.data
            this.$router.push('/')
          },
          (error) => {
            delete sessionStorage.loggedUser
            this.loggedUser = null
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.login'),
                autoHideDelay: 5000,
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    }
  }
}
</script>
