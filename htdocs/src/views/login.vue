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
      <b-col sm="4" class="d-none d-sm-flex align-items-center">
        <div>
          Don't already have an account on our beautiful platform?
          <router-link to="/join" variant="outline-primary" class="font-weight-bold">
            Join now!
          </router-link>
        </div>
      </b-col>
      <b-col sm="8">
        <b-card header-tag="div">
          <template v-slot:header>
            <h6 class="mb-0 font-weight-bold">
              Happy to see you again!
            </h6>
          </template>
          <form ref="form" @submit.stop.prevent="testlogin">
            <b-form-group
              :state="loginForm.emailState"
              label="Email address"
              label-for="email-input"
              invalid-feedback="Email address is required"
            >
              <b-form-input
                id="email-input"
                ref="loginemail"
                v-model="loginForm.email"
                :state="loginForm.emailState"
                required
                autofocus
                type="email"
                placeholder="pMockapetris@usc.edu"
                autocomplete="username"
              />
            </b-form-group>
            <b-form-group
              :state="loginForm.passwordState"
              label="Password"
              label-for="password-input"
              invalid-feedback="Password is required"
            >
              <b-form-input
                id="password-input"
                ref="loginpassword"
                v-model="loginForm.password"
                type="password"
                :state="loginForm.passwordState"
                required
                placeholder="xXxXxXxXxX"
                autocomplete="current-password"
              />
            </b-form-group>
            <div class="d-flex justify-content-around">
              <b-button type="submit" variant="primary">
                Go!
              </b-button>
              <b-button type="button" variant="outline-dark">
                Forgotten password?
              </b-button>
            </div>
          </form>
        </b-card>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
export default {

  data: function () {
    return {
      loginForm: {}
    }
  },

  methods: {
    testlogin () {
      const valid = this.$refs.form.checkValidity()
      this.loginForm.emailState = valid ? 'valid' : 'invalid'
      this.loginForm.passwordState = valid ? 'valid' : 'invalid'
      if (valid) {
        this.$parent.$emit('login', this.loginForm.email, this.loginForm.password)
      }
    }
  }
}
</script>
