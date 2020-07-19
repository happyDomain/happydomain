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
  <b-list-group>
    <b-list-group-item v-if="isLoading" class="d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" label="Spinning" class="mr-3" /> Retrieving your sources...
    </b-list-group-item>
    <b-list-group-item v-if="!isLoading && sources.length == 0" class="text-center">
      You have no source defined currently. Try <a href="#" @click.prevent="$emit('newSource')">adding one</a>!
    </b-list-group-item>
    <b-list-group-item v-for="(source, index) in sources" :key="index" button class="d-flex justify-content-between align-items-center" @click="selectSource(source)">
      <div>
        <div class="d-inline-block text-center" style="width: 50px;">
          <img v-if="sources_specs" :src="'/api/source_specs/' + source._srctype + '.png'" :alt="sources_specs[source._srctype].name" :title="sources_specs[source._srctype].name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
        </div>
        <span v-if="source._comment">{{ source._comment }}</span>
        <em v-else>No name</em>
      </div>
      <div>
        <b-badge class="ml-1" :variant="domain_in_sources[index] > 0 ? 'success' : 'danger'">
          {{ domain_in_sources[index] }} domain(s) associated
        </b-badge>
        <b-badge class="ml-1" variant="secondary" :title="source._srctype">
          {{ sources_specs[source._srctype].name }}
        </b-badge>
      </div>
    </b-list-group-item>
  </b-list-group>
</template>

<script>
import axios from 'axios'

export default {
  name: 'SourceList',

  props: {
    emitNewIfEmpty: {
      type: Boolean,
      default: false
    }
  },

  data: function () {
    return {
      domains: null,
      sources: null,
      sources_specs: null
    }
  },

  computed: {
    domain_in_sources () {
      var ret = {}

      if (this.domains != null && this.sources != null) {
        this.sources.forEach(function (source, idx) {
          ret[idx] = 0
          this.domains.forEach(function (domain) {
            if (domain.id_source === source._id) {
              ret[idx]++
            }
          })
        }, this)
      }

      return ret
    },

    isLoading () {
      return this.domains == null || this.sources == null || this.sources_specs == null
    }
  },

  mounted () {
    axios
      .get('/api/domains')
      .then(response => { this.domains = response.data })
    this.updateSources()
    axios
      .get('/api/source_specs')
      .then(response => { this.sources_specs = response.data })
  },

  methods: {
    selectSource (source) {
      this.$emit('sourceSelected', source)
    },

    updateSources () {
      axios
        .get('/api/sources')
        .then(response => {
          this.sources = response.data
          if (this.sources.length === 0 && this.emitNewIfEmpty) {
            this.$emit('newSource')
          }
        })
    }
  }
}
</script>
