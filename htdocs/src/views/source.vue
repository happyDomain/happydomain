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
  <b-container fluid class="mt-4">
    <h1 class="text-center mb-4">
      <button type="button" class="btn font-weight-bolder" @click="$router.go(-1)">
        <b-icon icon="chevron-left" />
      </button>
      Updating your domain name source <em v-if="mySource">{{ mySource._comment }}</em>
    </h1>
    <hr style="margin-bottom:0">

    <b-row style="min-height: inherit">
      <b-col v-if="source_specs_selected && sources" lg="4" md="5" class="bg-light">
        <div class="text-center mb-3">
          <img :src="'/api/source_specs/' + source_specs_selected + '.png'" :alt="sources[source_specs_selected].name" style="max-width: 100%; max-height: 10em">
        </div>
        <h3>
          {{ sources[source_specs_selected].name }}
        </h3>

        <p class="text-muted text-justify">
          {{ sources[source_specs_selected].description }}
        </p>

        <div class="text-center mb-2">
          <b-button v-if="source_specs && source_specs.capabilities && source_specs.capabilities.indexOf('ListDomains') > -1" type="button" variant="secondary" class="mb-1" @click="showListImportableDomain()">
            <b-icon icon="list-task" />
            List importable domains
          </b-button>
          <b-button type="button" variant="danger" class="mb-1" @click="deleteSource()">
            <b-icon icon="trash-fill" />
            Delete this source
          </b-button>
        </div>
      </b-col>

      <b-col lg="8" md="7">
        <router-view v-if="!isLoading" :parent-loading="isLoading" :my-source="mySource" :sources="sources" :source-specs="source_specs" :source-specs-selected="source_specs_selected" @updateMySource="updateMySource" />
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      mySource: null,
      sources: null,
      source_specs: null,
      source_specs_selected: null
    }
  },

  computed: {
    isLoading () {
      return this.mySource == null || this.sources == null || this.source_specs == null || this.source_specs_selected == null
    }
  },

  mounted () {
    axios
      .get('/api/sources/' + encodeURIComponent(this.$route.params.source))
      .then(response => {
        this.mySource = response.data
        this.source_specs_selected = this.mySource._srctype

        axios
          .get('/api/source_specs/' + encodeURIComponent(this.mySource._srctype))
          .then(response => {
            this.source_specs = response.data
            return true
          })

        return true
      })
    axios
      .get('/api/source_specs')
      .then(response => {
        this.sources = response.data
        return true
      })
  },

  methods: {
    deleteSource () {
      axios
        .delete('/api/sources/' + encodeURIComponent(this.$route.params.source))
        .then(
          response => {
            this.$router.push('/sources/')
          },
          error => {
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'Something went wrong during source deletion',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },
    showListImportableDomain () {
      this.$router.push('/sources/' + encodeURIComponent(this.$route.params.source) + '/domains')
    },
    updateMySource (newSource) {
      this.mySource = newSource
    }
  }
}
</script>
