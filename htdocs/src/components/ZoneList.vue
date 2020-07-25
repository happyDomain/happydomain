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
  <div>
    <b-list-group>
      <b-list-group-item v-if="isLoading" class="d-flex justify-content-center align-items-center">
        <b-spinner variant="primary" label="Spinning" class="mr-3" /> Retrieving your domains...
      </b-list-group-item>
      <b-list-group-item v-for="(domain, index) in sortedDomains" :key="index" :to="'/domains/' + domain.domain" class="d-flex justify-content-between align-items-center">
        <div class="text-monospace">
          <div class="d-inline-block text-center" style="width: 50px;">
            <img v-if="sources[domain.id_source]" :src="'/api/source_specs/' + sources[domain.id_source]._srctype + '/icon.png'" :alt="sources[domain.id_source]._srctype" :title="sources[domain.id_source]._srctype" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
          </div>
          {{ domain.domain }}
        </div>
        <b-badge variant="success">
          OK
        </b-badge>
      </b-list-group-item>
    </b-list-group>
    <h-list-group-input v-if="!isLoading" v-model="newDomain" autofocus class="mt-2" placeholder="my.new.domain." :state="newDomainState" input-class="text-monospace" @submit="submitNewDomain" @update="validateNewDomain" />
  </div>
</template>

<script>
import axios from 'axios'
import DomainCompare from '@/mixins/domainCompare'
import ListGroupInputNewDomain from '@/mixins/listGroupInputNewDomain'

export default {
  name: 'ZoneList',

  mixins: [DomainCompare, ListGroupInputNewDomain],

  data: function () {
    return {
      domains: null,
      sources: {}
    }
  },

  computed: {
    isLoading () {
      return this.domains == null
    },

    sortedDomains () {
      if (!this.domains) {
        return []
      }

      var ret = this.domains

      var vm = this
      ret.sort(function (a, b) { return vm.domainCompare(a.domain, b.domain) })

      return ret
    }
  },

  watch: {
    domains: function (domains) {
      this.$emit('noDomain', this.domains.length === 0)
    }
  },

  mounted () {
    axios
      .get('/api/domains')
      .then(response => {
        this.domains = response.data
        this.domains.forEach(function (domain) {
          axios
            .get('/api/sources/' + encodeURIComponent(domain.id_source))
            .then(response => {
              this.$set(this.sources, domain.id_source, response.data)
            })
        }, this)
      })
  },

  methods: {
    show (domain) {
      this.$router.push('/domains/' + encodeURIComponent(domain.domain))
    }
  }
}
</script>
