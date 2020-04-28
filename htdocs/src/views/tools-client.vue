<template>
<b-container class="mt-4 mb-5">
  <h1 class="text-center mb-4">
    DNS resolver
  </h1>
  <b-row class="mb-4">
    <b-col offset-md="2" md="8">
      <form @submit.stop.prevent="submitRequest" class="mb-5">

        <b-form-group
          id="input-resolver"
          label="Resolver"
          label-for="resolver"
          description="Give an explicit name in order to easily find this service."
          >
          <b-form-select
            id="resolver"
            v-model="form.resolver"
            required
            :options="existing_resolvers"
            ></b-form-select>
        </b-form-group>

        <b-form-group
          id="input-domain"
          label="Domain or subdomain"
          label-for="domain"
          description="spec.description"
          >
          <b-form-input
            id="domain"
            v-model="form.domain"
            required
            placeholder="happydns.org"
            ></b-form-input>
        </b-form-group>

        <b-form-group
          id="input-type"
          label="Field"
          label-for="type"
          description="spec.type"
          >
          <b-form-select
            id="type"
            v-model="form.type"
            required
            :options="existing_types"
            ></b-form-select>
        </b-form-group>

        <div class="ml-3 mr-3">
          <b-button class="float-right" type="submit" variant="primary" :disabled="request_pending">
            <b-spinner label="Spinning" small v-if="request_pending"></b-spinner>
            Run the request!
          </b-button>
        </div>
      </form>
    </b-col>
  </b-row>

  <b-row v-if="responses.length">
    <b-col offset-md="1" md="10">
      <b-alert v-for="(response,index) in responses" v-bind:key="index" v-model="show_responses[index]" variant="success" dismissible>
        <pre>{{response}}</pre>
      </b-alert>
    </b-col>
  </b-row>

</b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      request_pending: false,
      existing_types: ['ANY', 'A', 'AAAA', 'NS', 'SRV', 'MX', 'TXT'],
      existing_resolvers: ['192.168.0.254', '1.1.1.1', '8.8.8.8', '9.9.9.9'],
      form: {
        resolver: '192.168.0.254',
        type: 'ANY'
      },
      show_responses: [],
      responses: []
    }
  },

  mounted () {
    if (this.$route.params.domain) {
      this.form.type = 'A'
      this.form.domain = this.$route.params.domain
      this.submitRequest()
    }
  },

  methods: {
    submitRequest () {
      this.request_pending = true
      axios
        .post('/api/resolver', this.form)
        .then(
          (response) => {
            this.show_responses.unshift(true)
            this.responses.unshift(response.data)
            this.request_pending = false
          })
    }
  }
}
</script>

<style>
  .form-group label {
    font-weight: bold;
  }
</style>
