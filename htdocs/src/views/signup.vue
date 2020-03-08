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
            <h6 class="mb-0 font-weight-bold">Join our nice platform in less than 2 minutes!</h6>
          </template>
  <form class="container mt-2" @submit.stop.prevent="goSignUp" ref="form">
    <b-form-group
      :state="signupForm.emailState"
      label="Email address"
      label-for="email-input"
      invalid-feedback="Email address is required"
      >
      <b-form-input
        id="email-input"
        v-model="signupForm.email"
        :state="signupForm.emailState"
        required
        autofocus
        type="email"
        placeholder="jPostel@isi.edu"
        ref="signupemail"
        ></b-form-input>
    </b-form-group>
    <b-form-group
      :state="signupForm.passwordState"
      label="Password"
      label-for="password-input"
      invalid-feedback="Password is required"
      >
      <b-form-input
        type="password"
        id="password-input"
        v-model="signupForm.password"
        :state="signupForm.passwordState"
        required
        placeholder="xXxXxXxXxX"
        ref="signuppassword"
        ></b-form-input>
    </b-form-group>
    <b-form-group
      :state="signupForm.passwordConfirmState"
      label="Password confirmation"
      label-for="passwordconfirm-input"
      invalid-feedback="Password confirmation is required"
      >
      <b-form-input
        type="password"
        id="passwordconfirm-input"
        v-model="signupForm.passwordConfirm"
        :state="signupForm.passwordConfirmState"
        required
        placeholder="xXxXxXxXxX"
        ref="signuppasswordconfirm"
        ></b-form-input>
    </b-form-group>
    <div class="d-flex justify-content-around">
      <b-button type="submit" variant="success">Sign up!</b-button>
      <router-link to="/login" class="btn btn-outline-primary">Already member?</router-link>
    </div>
  </form>
        </b-card>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      signupForm: {}
    }
  },

  methods: {
    goSignUp () {
      const valid = this.$refs.form.checkValidity()
      this.signupForm.emailState = valid ? 'valid' : 'invalid'
      this.signupForm.passwordState = valid ? 'valid' : 'invalid'

      if (this.signupForm.password !== this.signupForm.passwordConfirm) {
        this.signupForm.passwordState = 'invalid'
        this.signupForm.passwordConfirmState = 'invalid'
      } else if (valid) {
        axios
          .post('/api/users', {
            'email': this.signupForm.email,
            'password': this.signupForm.password
          })
          .then(
            (response) => {
              alert('Registration successfully performed: userid=' + response.data.response.id)
              this.$router.push('/')
            },
            (error) => {
              alert('An error occurs when trying to register: ' + error.response.data.errmsg)
            }
          )
      }
    }
  }
}
</script>
