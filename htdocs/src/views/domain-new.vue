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
      Select the source where lives your domain <span class="text-monospace">{{ $route.params.domain }}</span>
    </h1>
    <hr style="margin-bottom:0">

    <b-row v-if="step === 0" class="mb-5">
      <b-col>
        <h3>
          Your existing sources
        </h3>

        <h-user-source-selector :sources="sources" @sourceSelected="selectExistingSource" />
      </b-col>
      <b-col lg="6">
        <h3>Use a new source</h3>

        <h-new-source-selector @sourceSelected="selectNewSource" />
      </b-col>
    </b-row>

    <div v-if="step & 1">
      <b-row>
        <b-col lg="4" md="5" class="bg-light">
          <div class="text-center mb-3 mt-2">
            <img :src="'/api/source_specs/' + source_specs_selected + '.png'" :alt="sources[source_specs_selected].name" style="max-width: 100%; max-height: 10em">
          </div>
          <h3>
            {{ sources[source_specs_selected].name }}
          </h3>

          <p class="text-muted text-justify">
            {{ sources[source_specs_selected].description }}
          </p>
        </b-col>

        <b-col lg="8" md="7">
          <form v-if="!isLoading" class="mt-2 mb-5" @submit.stop.prevent="submitNewSource">
            <h-resource-value-simple-input
              id="src-name"
              v-model="new_source_name"
              edit
              :index="0"
              label="Name your source"
              description="Give an explicit name in order to easily find this service."
              :placeholder="sources[source_specs_selected].name + ' 1'"
              required
            />

            <h-fields v-model="source_specs_values" edit :fields="source_specs.fields" />

            <div class="ml-3 mr-3">
              <b-button type="button" variant="secondary" @click="step=step&(~1)">
                &lt; Use another source
              </b-button>
              <b-button class="float-right" type="submit" variant="primary">
                Add this source &gt;
              </b-button>
            </div>
          </form>
        </b-col>
      </b-row>
    </div>

    <div v-if="step & 2 && isLoading" class="d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" label="Spinning" class="mr-3" /> Validating source...
    </div>
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  components: {
    hFields: () => import('@/components/hFields'),
    hNewSourceSelector: () => import('@/components/hNewSourceSelector'),
    hResourceValueSimpleInput: () => import('@/components/hResourceValueSimpleInput'),
    hUserSourceSelector: () => import('@/components/hUserSourceSelector')
  },

  data: function () {
    return {
      new_source_name: '',
      sources: null,
      source_specs: null,
      source_specs_selected: null,
      source_specs_values: {},
      step: 0
    }
  },

  computed: {
    isLoading () {
      if (this.step === 0) {
        return this.sources == null
      } else if (this.step & 1) {
        return this.source_specs_selected == null || this.source_specs == null
      } else if (this.step & 2) {
        return true
      } else {
        return false
      }
    }
  },

  mounted () {
    axios
      .get('/api/source_specs')
      .then(response => (this.sources = response.data))
  },

  methods: {

    selectNewSource (_, sourceSpec) {
      this.step |= 1
      this.source_specs_selected = sourceSpec
      axios
        .get('/api/source_specs/' + encodeURIComponent(sourceSpec))
        .then(
          response => {
            this.source_specs = response.data
          },
          error => {
            this.step &= ~1
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when creating the source!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    },

    selectExistingSource (source) {
      this.step |= 2

      axios
        .post('/api/domains', {
          id_source: source._id,
          domain: this.$route.params.domain
        })
        .then(
          (response) => {
            this.$root.$bvToast.toast(
              'Great! ' + response.data.domain + ' has been added. You can manage it right now.', {
                title: 'New domain attached to happyDNS!',
                autoHideDelay: 5000,
                variant: 'success',
                href: 'domains/' + encodeURIComponent(response.data.domain),
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/')
          },
          (error) => {
            this.step &= ~2
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when creating the source!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    },

    submitNewSource () {
      var mySource = {
        _srctype: this.source_specs_selected,
        _comment: this.new_source_name,
        Source: this.source_specs_values
      }

      axios
        .post('/api/sources', mySource)
        .then(
          (response) => {
            this.selectExistingSource(response.data)
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when creating the source!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    }

  }
}
</script>
