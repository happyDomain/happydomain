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
    <b-list-group-item v-if="!isLoading && sources_getAll.length == 0" class="text-center">
      You have no source defined currently. Try <a href="#" @click.prevent="$emit('new-source')">adding one</a>!
    </b-list-group-item>
    <b-list-group-item v-for="(source, index) in sortedSources" :key="index" :active="selectedSource && selectedSource._id === source._id" button class="d-flex justify-content-between align-items-center" @click="selectSource(source)">
      <div class="d-flex">
        <div class="text-center" style="width: 50px;">
          <img v-if="sourceSpecs_getAll" :src="'/api/source_specs/' + source._srctype + '/icon.png'" :alt="sourceSpecs_getAll[source._srctype].name" :title="sourceSpecs_getAll[source._srctype].name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
        </div>
        <div v-if="source._comment" style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
          {{ source._comment }}
        </div>
        <em v-else>No name</em>
      </div>
      <div v-if="!(noLabel && noDropdown)" class="d-flex">
        <div v-if="!noLabel">
          <b-badge class="ml-1" :variant="domain_in_sources[source._id] > 0 ? 'success' : 'danger'">
            {{ domain_in_sources[source._id] }} domain(s) associated
          </b-badge>
          <b-badge v-if="sourceSpecs_getAll" class="ml-1" variant="secondary" :title="source._srctype">
            {{ sourceSpecs_getAll[source._srctype].name }}
          </b-badge>
        </div>
        <b-dropdown v-if="!noDropdown" no-caret size="sm" style="margin-right: -10px" variant="link">
          <template #button-content>
            <b-icon icon="three-dots" />
          </template>
          <b-dropdown-item @click="updateSource($event, source)">
            Update settings
          </b-dropdown-item>
          <b-dropdown-item @click="deleteSource($event, source)">
            Delete source
          </b-dropdown-item>
        </b-dropdown>
      </div>
    </b-list-group-item>
  </b-list-group>
</template>

<script>
import { mapGetters } from 'vuex'

import SourceApi from '@/services/SourcesApi'

export default {
  name: 'SourceList',

  props: {
    emitNewIfEmpty: {
      type: Boolean,
      default: false
    },
    noDropdown: {
      type: Boolean,
      default: false
    },
    noLabel: {
      type: Boolean,
      default: false
    }
  },

  data: function () {
    return {
      selectedSource: null
    }
  },

  computed: {
    domain_in_sources () {
      const ret = {}

      if (this.sources_getAll != null) {
        for (const i in this.sources_getAll) {
          ret[i] = 0
        }
      }

      if (this.domains_getAll != null) {
        this.domains_getAll.forEach(function (domain) {
          if (!ret[domain.id_source]) {
            ret[domain.id_source] = 0
          }
          ret[domain.id_source]++
        })
      }

      return ret
    },

    isLoading () {
      return (!this.noLabel && this.domains_getAll == null) || this.sources_getAll == null || this.sourceSpecs_getAll == null
    },

    ...mapGetters('domains', ['domains_getAll']),
    ...mapGetters('sources', ['sortedSources', 'sources_getAll']),
    ...mapGetters('sourceSpecs', ['sourceSpecs_getAll'])
  },

  watch: {
    sources_getAll: function (sources) {
      if (Object.keys(sources).length === 0 && this.emitNewIfEmpty) {
        this.$emit('new-source')
      }
    }
  },

  methods: {
    deleteSource (event, source) {
      event.stopPropagation()
      SourceApi.deleteSource(source)
        .then(
          response => {
            this.$store.dispatch('sources/getAllMySources')
            this.$bvToast.toast(
              'The source has been deleted with success.', {
                title: 'Source deleted with success',
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              })
          },
          error => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when trying to delete the source:',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              })
          })
    },

    selectSource (source) {
      if (this.selectedSource != null && this.selectedSource._id === source._id) {
        this.selectedSource = null
      } else {
        this.selectedSource = source
      }
      this.$emit('source-selected', this.selectedSource)
    },

    updateSource (event, source) {
      event.stopPropagation()
      this.$router.push('/sources/' + encodeURIComponent(source._id))
    }
  }
}
</script>
