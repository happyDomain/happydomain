<template>
<b-container class="mt-4 mb-5">
  <h1 class="text-center mb-4">
    Domains living on <span v-if="mySource">{{ mySource._comment }}</span>
  </h1>
  <b-row>
    <b-col offset-md="2" md="8">
      <div class="text-right">
      </div>
      <b-list-group>
        <b-list-group-item v-if="isLoading" class="text-center">
          <b-spinner variant="secondary" label="Spinning"></b-spinner> Asking provider for the existing domains...
        </b-list-group-item>
        <b-list-group-item v-for="(domain, index) in domains" v-bind:key="index" class="d-flex justify-content-between align-items-center">
          <div>
            {{ domain }}
          </div>
          <div>
            <b-badge class="ml-1" variant="success" v-if="myDomains.indexOf(domain) > -1"><b-icon icon="check" /> Already managed</b-badge>
            <b-button type="button" class="ml-1" variant="primary" size="sm" @click="importDomain(domain)" v-else>Add now</b-button>
          </div>
        </b-list-group-item>
      </b-list-group>
    </b-col>
  </b-row>
</b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      myDomains: null,
      domains: null
    }
  },

  mounted () {
    axios
      .get('/api/sources/' + this.$route.params.source + '/domains')
      .then(response => (this.domains = response.data))
    axios
      .get('/api/domains')
      .then(response => {
        var domains = []
        response.data.forEach(function (domain) {
          domains.push(domain.domain)
        })
        this.myDomains = domains
      })
  },

  computed: {
    isLoading () {
      return this.parentLoading || this.domains == null || this.myDomains == null
    }
  },

  methods: {
    importDomain (domain) {
      axios
        .post('/api/domains', {
          id_source: this.mySource._id,
          domain: domain
        })
        .then(
          (response) => {
            this.$bvToast.toast(
              'Great! ' + response.data.domain + ' has been added. You can manage it right now.', {
                title: 'New domain attached to happyDNS!',
                autoHideDelay: 5000,
                variant: 'success',
                href: 'domains/' + response.data.domain,
                toaster: 'b-toaster-content-right'
              }
            )
            this.myDomains.push(response.data.domain)
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when attaching the domain to happyDNS',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    }
  },

  props: ['parentLoading', 'mySource']
}
</script>
