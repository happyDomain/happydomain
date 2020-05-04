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
    <b-navbar style="border-bottom: 3px solid #aee64e; box-shadow: 0 0 12px 0 #08334833; z-index:2">
      <b-navbar-brand class="navbar-brand" to="/" style="font-family: 'Fortheenas01';font-weight:bold;">
        happy<span style="font-family: 'Fortheenas01 Bold';margin-left:.1em;">DNS</span>
      </b-navbar-brand>
      <button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#adminMenu" aria-controls="adminMenu" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
      </button>

      <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>

      <b-navbar-nav class="ml-auto">
        <b-nav-item-dropdown right v-if="loggedUser">
          <template slot="button-content"><div class="btn btn-sm btn-secondary"><b-icon icon="person" aria-hidden="true"></b-icon> {{ loggedUser.email }}</div></template>
          <b-dropdown-item to="/domains/">My domains</b-dropdown-item>
          <b-dropdown-item to="/sources/">My sources</b-dropdown-item>
          <b-dropdown-divider></b-dropdown-divider>
          <b-dropdown-item to="/tools/client">DNS client</b-dropdown-item>
          <b-dropdown-divider></b-dropdown-divider>
          <b-dropdown-item @click="logout()">Logout</b-dropdown-item>
        </b-nav-item-dropdown>
        <b-button v-if="!loggedUser" variant="outline-success" @click="signup()"><b-icon icon="person-fill" aria-hidden="true"></b-icon> Sign up</b-button>
        <b-button v-if="!loggedUser" variant="primary" class="ml-2" @click="signin()"><b-icon icon="person-fill" aria-hidden="true"></b-icon> Sign in</b-button>
      </b-navbar-nav>
    </b-navbar>

    <router-view/>

    <b-toaster name="b-toaster-content-right" style="position: fixed; top: 70px; right: 0; z-index: 10; min-width: 30vw;"></b-toaster>

    <div class="pt-3 pb-5 bg-dark text-light" style="border-top: 3px solid #aee64e; box-shadow: 0 0 12px 0 #08334833; z-index: 2">
      <b-container>
        <b-row>
          <b-col md="4">
            &copy; <span style="font-family: 'Fortheenas01';font-weight:bold;">happy<span style="font-family: 'Fortheenas01 Bold';margin-left:.1em;">DNS</span></span> 2019-2020 All rights reserved
          </b-col>
          <b-col md="4">
          </b-col>
          <b-col md="4">
          </b-col>
        </b-row>
      </b-container>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

function updateSession (t) {
  if (sessionStorage.token !== undefined) {
    t.session = sessionStorage.token
    axios.defaults.headers.common.Authorization = 'Bearer '.concat(sessionStorage.token)
    axios.get('/api/users/auth')
      .then(
        (response) => {
          t.loggedUser = response.data
        },
        (error) => {
          t.$bvToast.toast(
            'Invalid session, your have been logged out: ' + error.response.data.errmsg + '. Please login again.', {
              title: 'Authentication timeout',
              autoHideDelay: 5000,
              variant: 'danger',
              toaster: 'b-toaster-content-right'
            }
          )
          t.session = null
          t.loggedUser = null
          delete sessionStorage.token
          t.$router.replace('/login')
        }
      )
  }
}

export default {

  data: function () {
    return {
      loggedUser: null,
      session: null
    }
  },

  mounted () {
    updateSession(this)
    this.$on('login', this.login)
  },

  methods: {
    signin () {
      this.$router.push('/login')
    },
    signup () {
      this.$router.push('/join')
    },

    logout () {
      sessionStorage.token = undefined
      updateSession(this)
      this.$router.push('/')
    },

    login (email, password) {
      axios
        .post('/api/users/auth', {
          email: email,
          password: password
        })
        .then(
          (response) => {
            if (response.data.id_session) {
              sessionStorage.token = response.data.id_session
            }
            updateSession(this)
            this.$router.push('/')
          },
          (error) => {
            this.$bvToast.toast(
              'An error occurs when trying to login: ' + error.response.data.errmsg, {
                title: 'Login error',
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
