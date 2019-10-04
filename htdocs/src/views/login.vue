<template>
  <form class="container mt-2" @submit.stop.prevent="testlogin" ref="form">
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
        ref="loginpassword"
        ></b-form-input>
    </b-form-group>
    <b-button type="submit" variant="success"><span class="glyphicon glyphicon-user" aria-hidden="true"></span> Go!</b-button>
  </form>
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
