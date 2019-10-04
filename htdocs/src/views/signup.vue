<template>
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
        ref="signuppasswordconfirm"
        ></b-form-input>
    </b-form-group>
    <b-button type="submit" variant="success"><span class="glyphicon glyphicon-user" aria-hidden="true"></span> Sign up!</b-button>
  </form>
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
