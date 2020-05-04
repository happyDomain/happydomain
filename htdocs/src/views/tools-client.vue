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
