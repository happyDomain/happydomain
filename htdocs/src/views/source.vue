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
  <b-container class="d-flex flex-column mt-4" fluid>
    <h1 class="text-center mb-3">
      <button type="button" class="btn font-weight-bolder" @click="$router.go(-1)">
        <b-icon icon="chevron-left" />
      </button>
      {{ $t('wait.updating') }} <em v-if="mySource">{{ mySource._comment }}</em>
    </h1>
    <hr class="mt-0 mb-0">

    <b-row class="flex-grow-1">
      <b-col v-if="sourceSpecsSelected && sources" lg="4" md="5" class="bg-light">
        <div class="text-center mb-3">
          <img :src="'/api/source_specs/' + sourceSpecsSelected + '/icon.png'" :alt="sources[sourceSpecsSelected].name" style="max-width: 100%; max-height: 10em">
        </div>
        <h3>
          {{ sources[sourceSpecsSelected].name }}
        </h3>

        <p class="text-muted text-justify">
          {{ sources[sourceSpecsSelected].description }}
        </p>

        <div class="text-center mb-2">
          <b-button type="button" variant="danger" class="mb-1" @click="deleteSource()">
            <b-icon icon="trash-fill" />
            {{ $t('source.delete') }}
          </b-button>
        </div>
      </b-col>

      <b-col lg="8" md="7">
        <b-form v-if="!isLoading" class="mt-2 mb-5" @submit.stop.prevent="nextState">
          <h-source-state
            v-if="form"
            v-model="settings"
            class="mt-2 mb-2"
            :form="form"
            :source-name="sources[sourceSpecsSelected].name"
            :state="state"
          />

          <hr>

          <h-source-state-buttons v-if="form" class="d-flex justify-content-end" edit :form="form" :next-is-working="nextIsWorking" :previous-is-working="previousIsWorking" @previous-state="previousState" />
        </b-form>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import SourceState from '@/mixins/sourceState'
import axios from 'axios'

export default {

  components: {
    hSourceState: () => import('@/components/hSourceState'),
    hSourceStateButtons: () => import('@/components/hSourceStateButtons')
  },

  mixins: [SourceState],

  data: function () {
    return {
      mySource: null,
      sources: null,
      sourceSpecs: null,
      sourceSpecsSelected: null
    }
  },

  computed: {
    isLoadingHere () {
      return this.mySource == null || this.sources == null || this.sourceSpecs == null || this.sourceSpecsSelected == null || this.settings == null || this.isLoading
    }
  },

  mounted () {
    this.resetSettings()
    axios
      .get('/api/sources/' + encodeURIComponent(this.$route.params.source))
      .then(response => {
        this.mySource = response.data
        this.settings = this.mySource
        this.sourceSpecsSelected = this.mySource._srctype

        axios
          .get('/api/source_specs/' + encodeURIComponent(this.mySource._srctype))
          .then(response => {
            this.sourceSpecs = response.data

            this.loadState(0)
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
                title: this.$t('errors.source-delete'),
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
    reactOnSuccess (toState, newSource) {
      if (newSource) {
        this.mySource = newSource
      }
    }
  }
}
</script>
