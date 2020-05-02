<template>
<b-container class="mt-4 mb-5">
  <h1 class="text-center mb-4">Welcome to <span style="font-family: 'Fortheenas01';font-weight:bold;">happy<span style="font-family: 'Fortheenas01 Bold';margin-left:.1em;">DNS</span></span>!</h1>
  <b-row>
    <div class="offset-md-2 col-md-8">
      <b-list-group>
        <b-list-group-item v-if="isLoading" class="text-center">
          <b-spinner variant="secondary" label="Spinning"></b-spinner> Retrieving your domains...
        </b-list-group-item>
        <b-list-group-item :to="'/domains/' + domain.domain" v-for="(domain, index) in domains" v-bind:key="index" class="d-flex justify-content-between align-items-center">
          {{ domain.domain }}
          <b-badge variant="success">OK</b-badge>
        </b-list-group-item>
      </b-list-group>
      <b-list-group class="mt-2">
        <form @submit.stop.prevent="submitNewDomain" v-if="!isLoading">
          <b-list-group-item class="d-flex justify-content-between align-items-center">
            <b-input-group>
              <template v-slot:prepend>
                <b-input-group-prepend @click="$refs.newdomain.focus()">
                  <b-icon icon="plus" style="width: 2.3em; height: 2.3rem; margin-left: -.5em"></b-icon>
                </b-input-group-prepend>
              </template>
              <b-form-input placeholder="my.new.domain" ref="newdomain" v-model="newDomain" @update="validateNewDomain" :state="newDomainState" style="border:none;box-shadow:none;z-index:0"></b-form-input>
              <template v-slot:append>
                <b-input-group-append v-show="newDomain.length">
                  <b-button type="submit" variant="outline-primary">Add new domain</b-button>
                </b-input-group-append>
              </template>
            </b-input-group>
          </b-list-group-item>
        </form>
      </b-list-group>
    </div>
  </b-row>
</b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      newDomain: '',
      newDomainState: null,
      domains: null
    }
  },

  mounted () {
    setTimeout(() =>
      axios
        .get('/api/domains')
        .then(response => (this.domains = response.data))
    , 100)
  },

  computed: {
    isLoading () {
      return this.domains == null
    }
  },

  methods: {
    show (domain) {
      this.$router.push('/domains/' + domain.domain)
    },

    validateNewDomain () {
      if (this.newDomain.length === 0) {
        this.newDomainState = null
      } else {
        this.newDomainState = this.newDomain.length >= 4 && this.newDomain.length <= 254

        if (this.newDomainState) {
          var domains = this.newDomain.split('.')

          // Remove the last . if any, it's ok
          if (domains[domains.length - 1] === '') {
            domains.pop()
          }

          var newDomainState = this.newDomainState
          domains.forEach(function (domain) {
            newDomainState &= domain.length >= 1 && domain.length <= 63
            newDomainState &= domain[0] !== '-' && domain[domain.length - 1] !== '-'
            newDomainState &= /^[a-zA-Z0-9]([a-zA-Z0-9-]?[a-zA-Z0-9])*$/.test(domain)
          })
          this.newDomainState = newDomainState > 0
        }
      }

      return this.newDomainState
    },

    submitNewDomain () {
      if (this.validateNewDomain()) {
        this.$router.push('/domains/' + this.newDomain + '/new')
      }
    }

  }
}
</script>
