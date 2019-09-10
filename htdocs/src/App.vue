<template>
  <div id="app">
    <b-navbar size="lg" type="dark" variant="dark" sticky class="text-light">
      <b-navbar-brand class="navbar-brand" to="/">
        <img alt="LibreDNS" src="<%= BASE_URL %>img/logo.png" style="height: 30px">
      </b-navbar-brand>
      <button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#adminMenu" aria-controls="adminMenu" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
      </button>

      <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>

      <b-collapse id="nav-collapse" is-nav>
        <b-navbar-nav>
          <b-nav-item to="/zones">Zones</b-nav-item>
          <b-nav-item to="/users" disabled>Users</b-nav-item>
        </b-navbar-nav>
      </b-collapse>

      <b-navbar-nav class="ml-auto">
        <b-nav-item-dropdown right v-if="loggedUser">
          <template slot="button-content"><div class="btn btn-sm btn-secondary">{{ loggedUser.email }}</div></template>
          <b-dropdown-item>Some example text that's free-flowing within the dropdown menu.</b-dropdown-item>
          <b-dropdown-item href="#">Action</b-dropdown-item>
          <b-dropdown-item href="#">Another action</b-dropdown-item>
          <b-dropdown-item @click="logout()">Logout</b-dropdown-item>
        </b-nav-item-dropdown>
        <b-button v-if="!loggedUser" variant="success" @click="signup()"><span class="glyphicon glyphicon-user" aria-hidden="true"></span> Sign up</b-button>
        <b-button v-if="!loggedUser" variant="primary" class="ml-2" @click="signin()"><span class="glyphicon glyphicon-user" aria-hidden="true"></span> Sign in</b-button>
      </b-navbar-nav>
    </b-navbar>
    <div class="progress" style="background-color: #aee64e; height: 3px; border-radius: 0;">
      <div class="progress-bar bg-secondary" role="progressbar" style="width: 0%"></div>
    </div>

    <router-view/>
  </div>
</template>

<script>
import axios from 'axios'

function updateSession (t) {
  if (sessionStorage.token !== undefined) {
    t.session = sessionStorage.token
    axios.get('/api/users/auth', {
      headers: {
        'Authorization': 'Bearer '.concat(t.session)
      }
    })
      .then(
        (response) => {
          t.loggedUser = response.data
        },
        (error) => {
          console.error('Invalid session, your have been logged out:', error.response.errmsg)
          t.session = null
          t.loggedUser = null
          sessionStorage.token = undefined
          t.$router.push('/')
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
    this.$on('login', this.login)
    updateSession(this)
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
    },

    login (email, password) {
      axios
        .post('/api/users/auth', {
          'email': email,
          'password': password
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
            alert('An error occurs when trying to login: ' + error.response.data.errmsg)
          }
        )
    }
  }
}
</script>
