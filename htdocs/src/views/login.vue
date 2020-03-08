<template>
  <b-container style="margin-top: 10vh; margin-bottom: 10vh;">
    <b-row>
      <b-col sm="4" class="d-none d-sm-flex align-items-center">
        <div>
          Don't already have an account on our beautiful platform?
          <router-link to="/join" variant="outline-primary" class="font-weight-bold">Join now!</router-link>
        </div>
      </b-col>
      <b-col sm="8">
        <b-card header-tag="div">
          <template v-slot:header>
            <h6 class="mb-0 font-weight-bold">Happy to see you again!</h6>
          </template>
        <form @submit.stop.prevent="testlogin" ref="form">
          <b-form-group
            :state="loginForm.emailState"
            label="Email address"
            label-for="email-input"
            invalid-feedback="Email address is required"
            >
            <b-form-input
              id="email-input"
              v-model="loginForm.email"
              :state="loginForm.emailState"
              required
              autofocus
              type="email"
              placeholder="pMockapetris@usc.edu"
              ref="loginemail"
              ></b-form-input>
          </b-form-group>
          <b-form-group
            :state="loginForm.passwordState"
            label="Password"
            label-for="password-input"
            invalid-feedback="Password is required"
            >
            <b-form-input
              type="password"
              id="password-input"
              v-model="loginForm.password"
              :state="loginForm.passwordState"
              required
              placeholder="xXxXxXxXxX"
              ref="loginpassword"
              ></b-form-input>
          </b-form-group>
          <div class="d-flex justify-content-around">
            <b-button type="submit" variant="success">Go!</b-button>
            <b-button type="button" variant="outline-primary">Forgotten password?</b-button>
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
