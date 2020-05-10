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
    <div v-for="(dn, index) in sortedDomains" :key="index">
      <h2 :id="dn" :class="index !== 0 ? 'mt-4' : ''">
        <b-icon v-if="hideDomain[index]" icon="chevron-right" @click="toogleHideDomain(index)" />
        <b-icon v-if="!hideDomain[index]" icon="chevron-down" @click="toogleHideDomain(index)" />
        <span class="text-monospace" @click="toogleHideDomain(index)">{{ dn }}</span>
        <a :href="'#' + dn" class="float-right">
          <b-icon icon="link45deg" />
        </a>
        <b-badge class="ml-2" v-if="myServices.aliases && myServices.aliases[dn]" v-b-popover.hover.focus="{ customClass: 'text-monospace', html: true, content: myServices.aliases[dn].map(function(alias) { return escapeHTML(alias) }).join('<br>') }">+ {{ myServices.aliases[dn].length }} aliases</b-badge>
        <b-button type="button" variant="primary" size="sm" class="ml-2">
          <b-icon icon="plus" />
          Add a service
        </b-button>
        <b-button type="button" variant="outline-primary" size="sm" class="ml-2">
          <b-icon icon="link" />
          Add an alias
        </b-button>
      </h2>
      <b-list-group v-show="!hideDomain[index]" v-for="(svc, idx) in myServices.services[dn]" :key="idx">
        <b-list-group-item @click="toogleRR(index, idx)" button>
          <strong>{{ services[svc._svctype].name }}</strong> <span v-if="svc._comment" class="text-muted">{{ svc._comment }}</span>
          <span class="text-muted" v-if="services[svc._svctype].comment">{{ services[svc._svctype].comment }}</span>
          <b-badge v-for="(categorie, idcat) in services[svc._svctype].categories" :key="idcat" variant="gray" class="float-right ml-1">{{ categorie }}</b-badge>
          <b-badge v-if="svc._tmp_hint_nb && svc._tmp_hint_nb > 1" variant="danger" class="float-right ml-1">{{ svc._tmp_hint_nb }}</b-badge>
        </b-list-group-item>
        <b-list-group-item v-if="expandrrs['' + index + '.' + idx]">
          {{ svc }}
        </b-list-group-item>
      </b-list-group>
    </div>
  </div>
</template>

<script>
import axios from 'axios'
import Vue from 'vue'

export default {
  props: {
    domain: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      expandrrs: {},
      hideDomain: {},
      myServices: null,
      services: null
    }
  },

  computed: {
    isLoading () {
      return this.myServices == null || this.services == null
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
    $route: 'fetchData',

    domain: function () {
      this.pullDomain()
    }
  },

  created () {
    this.fetchData()
  },

  methods: {
    fetchData () {
      if (this.domain !== undefined && Object.keys(this.domain).length !== 0) {
        this.pullDomain()
      }

      axios
        .get('/api/services')
        .then(
          (response) => {
            this.services = response.data
            this.goToAnchor()
          }
        )
    },

    toogleHideDomain (idx) {
      Vue.set(this.hideDomain, idx, !this.hideDomain[idx])
    },

    toogleRR (index, idx) {
      Vue.set(this.expandrrs, index + '.' + idx, !this.expandrrs[index + '.' + idx])
    },

    goToAnchor () {
      var hash = this.$route.hash.substr(1)
      if (!this.isLoading && hash.length > 0) {
        setTimeout(function () {
          window.scrollTo(0, document.getElementById(hash).offsetTop)
        }, 500)
      }
    },

    pullDomain () {
      axios
        .post('/api/domains/' + this.domain.domain + '/analyze')
        .then(
          (response) => {
            this.myServices = response.data
            this.goToAnchor()
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
            this.$router.push('/domains/' + this.domain.domain)
          }
        )
    }
  }
}
</script>

<style scoped>
.services {
    display: flex;
    align-items: center;
    justify-content: center;
}
.service {
    box-shadow: 2px 2px black;
    border: 1px solid black;
    display: flex;
    flex-direction: column;
    align-items: center;
    margin: 2.5%;
    width: 20%;
    height: 150px;
    text-align: center;
    vertical-align: middle;
}
.service img {
    max-width: 100%;
    max-height: 90%;
    padding: 2%;
}
</style>
