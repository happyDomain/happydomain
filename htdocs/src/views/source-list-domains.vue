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
    <h3 class="text-center mb-4">
      Domains living on <em v-if="mySource">{{ mySource._comment }}</em>
    </h3>
    <b-list-group>
      <b-list-group-item v-if="isLoading" class="text-center">
        <b-spinner variant="secondary" label="Spinning" /> Asking provider for the existing domains...
      </b-list-group-item>
      <b-list-group-item v-for="(domain, index) in domainsList" :key="index" class="d-flex justify-content-between align-items-center" :to="haveDomain(domain) ? '/domains/' + encodeURIComponent(domain) : ''">
        <div>
          {{ domain }}
        </div>
        <div>
          <b-badge v-if="haveDomain(domain)" class="ml-1" variant="success">
            <b-icon icon="check" /> Already managed
          </b-badge>
          <b-button v-else type="button" class="ml-1" variant="primary" size="sm" @click="importDomain(domain)">
            Add now
          </b-button>
        </div>
      </b-list-group-item>
      <b-list-group-item v-if="!noDomainsList && !isLoading && domainsList.length === 0" class="text-center">
        It appears you don't have any domain name registered on this provider.
      </b-list-group-item>
      <b-list-group-item v-else-if="noDomainsList && !isLoading && domainsList.length === 0" class="text-center">
        This provider doesn't permit to list existing domains. Use the form below to add one.
      </b-list-group-item>
    </b-list-group>
    <h-list-group-input v-if="noDomainsList" v-model="newDomain" autofocus class="mt-2" placeholder="my.new.domain." :state="newDomainState" input-class="text-monospace" @submit="submitNewDomain" @update="validateNewDomain" />
  </b-container>
</template>

<script>
import axios from 'axios'
import ListGroupInputNewDomain from '@/mixins/listGroupInputNewDomain'

export default {

  mixins: [ListGroupInputNewDomain],

  props: {
    parentLoading: {
      type: Boolean,
      required: true
    },
    mySource: {
      type: Object,
      required: true
    },
    sourceSpecs: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      domains: null,
      myDomains: null,
      noDomainsList: false
    }
  },

  computed: {
    domainsList () {
      if (this.isLoading && !this.mySource) {
        return []
      } else if (this.noDomainsList) {
        var ret = []

        for (const i in this.myDomains) {
          if (this.myDomains[i].id_source === this.mySource._id) {
            ret.push(this.myDomains[i].domain)
          }
        }

        return ret
      } else {
        return this.domains
      }
    },
    isLoading () {
      return this.parentLoading || (this.domains == null && !this.noDomainsList) || this.myDomains == null
    }
  },

  watch: {
    sourceSpecs: function () {
      if (this.sourceSpecs) {
        this.listImportableDomains()
      }
    }
  },

  mounted () {
    if (this.sourceSpecs) {
      this.listImportableDomains()
    }
    this.refreshDomains()
  },

  methods: {
    haveDomain (domain) {
      for (const i in this.myDomains) {
        if (this.myDomains[i].domain === domain) {
          return true
        }
      }
      return false
    },

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
                href: '/domains/' + response.data.domain,
                toaster: 'b-toaster-content-right'
              }
            )
            this.myDomains.push(response.data)
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
    },

    listImportableDomains () {
      if (!this.sourceSpecs.capabilities || this.sourceSpecs.capabilities.indexOf('ListDomains') === -1) {
        this.noDomainsList = true
        return
      }

      axios
        .get('/api/sources/' + encodeURIComponent(this.$route.params.source) + '/domains')
        .then(
          response => (this.domains = response.data),
          error => {
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when trying to access domain\'s list.',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.replace('/sources/' + encodeURIComponent(this.mySource._id))
          })
    },

    refreshDomains () {
      this.myDomains = null
      this.newDomain = ''
      axios
        .get('/api/domains')
        .then(response => {
          this.myDomains = response.data
        })
    }
  }
}
</script>
