<template>
  <div class="container mt-2">
    <b-button variant="success" @click="testlogin()"><span class="glyphicon glyphicon-user" aria-hidden="true"></span> Go!</b-button>
  </div>
</template>

<script>
export default {

  methods: {
    testlogin () {
      this.$parent.$emit('login', 'nemunaire@nemunai.re', 'ciphuj5O')
    }
  }
}
</script>
