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
  <div v-if="!isLoading" class="pt-3">
    <h-subdomain-item v-for="(dn, index) in sortedDomains" :key="index" :dn="dn" :services="myServices.services[dn]===undefined?[]:myServices.services[dn]" :aliases="myServices.aliases[dn]===undefined?[]:myServices.aliases[dn]" />
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'HSubdomainList',

  components: {
    hSubdomainItem: () => import('@/components/hSubdomainItem')
  },

  props: {
    domain: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      hideDomain: {},
      myServices: null
    }
  },

  computed: {
    isLoading () {
      return this.myServices == null
    },

    sortedDomains () {
      if (this.myServices === null) {
        return []
      }

      var domains = Object.keys(this.myServices.services)
      domains.sort(function (a, b) {
        var as = a.split('.').reverse()
        var bs = b.split('.').reverse()

        var maxDepth = Math.min(as.length, bs.length)
        for (var i = 0; i < maxDepth; i++) {
          var cmp = as[i].localeCompare(bs[i])
          if (cmp !== 0) {
            return cmp
          }
        }

        return as.length - bs.length
      })

      return domains
    }
  },

  watch: {
    domain: function () {
      this.pullDomain()
    }
  },

  created () {
    if (this.domain !== undefined) {
      this.pullDomain()
    }
  },

  methods: {
    goToAnchor () {
      var hash = this.$route.hash.substr(1)
      if (!this.isLoading && hash.length > 0) {
        setTimeout(function () {
          window.scrollTo(0, document.getElementById(hash).offsetTop)
        }, 500)
      }
    },

    pullDomain () {
      if (this.domain === undefined || this.domain.domain === undefined) {
        return
      }
      axios
        .post('/api/domains/' + encodeURIComponent(this.domain.domain) + '/analyze')
        .then(
          (response) => {
            this.myServices = response.data
            // this.goToAnchor()
          },
          (error) => {
            this.$root.$bvToast.toast(
              'Unfortunately, we were unable to retrieve information for the domain ' + this.domain.domain + ': ' + error.response.data.errmsg, {
                title: 'Unable to retrieve domain information',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/domains/' + encodeURIComponent(this.domain.domain))
          }
        )
    }
  }
}
</script>
