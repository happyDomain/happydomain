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
  <b-container fluid>
    <div v-if="isLoading" class="mt-5 d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" label="Spinning" class="mr-3" /> Retrieving the source settings' form...
    </div>
    <b-row v-else>
      <b-col lg="4" md="5" class="bg-light">
        <div class="text-center mb-3">
          <img :src="'/api/source_specs/' + $route.params.provider + '.png'" :alt="sourceSpecs[$route.params.provider].name" style="max-width: 100%; max-height: 10em">
        </div>
        <h3>
          {{ sourceSpecs[$route.params.provider].name }}
        </h3>

        <p class="text-muted text-justify">
          {{ sourceSpecs[$route.params.provider].description }}
        </p>

        <hr v-if="form.sideText">
        <p v-if="form.sideText" class="text-justify">
          {{ form.sideText }}
        </p>
      </b-col>

      <b-col lg="8" md="7">
        <form class="mt-2 mb-5" @submit.stop.prevent="submitSettings">
          <p v-if="form.beforeText" class="lead text-indent">
            {{ form.beforeText }}
          </p>

          <h-resource-value-simple-input
            v-if="$route.params.state === '0'"
            id="src-name"
            v-model="mySrcName"
            edit
            :index="0"
            label="Name your source"
            description="Give an explicit name in order to easily find this service."
            :placeholder="sourceSpecs[$route.params.provider].name + ' 1'"
            required
          />

          <h-fields v-if="form.fields" v-model="settings" edit :fields="form.fields" />

          <p v-if="form.afterText">
            {{ form.afterText }}
          </p>

          <div class="d-flex justify-content-end">
            <b-button v-if="form.previousButtonText" type="button" variant="outline-secondary" class="mx-1" @click="previousState()">
              {{ form.previousButtonText }}
            </b-button>
            <b-button v-if="form.nextButtonText" type="button" variant="primary" class="mx-1" @click="submitSettings()">
              {{ form.nextButtonText }}
            </b-button>
          </div>
        </form>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import SourceSettingsApi from '@/services/SourceSettingsApi'
import SourceSpecsApi from '@/services/SourceSpecsApi'

export default {

  components: {
    hFields: () => import('@/components/hFields'),
    hResourceValueSimpleInput: () => import('@/components/hResourceValueSimpleInput')
  },

  data: function () {
    return {
      form: null,
      mySrcName: '',
      settings: {},
      sourceSpecs: null
    }
  },

  computed: {
    isLoading () {
      return this.form == null || this.sourceSpecs == null
    }
  },

  mounted () {
    this.updateSourceSettingsForm()
  },

  methods: {
    loadState (toState, settings, name, recallid) {
      var mySource = this.$route.params.provider
      SourceSettingsApi.getSourceSettings(mySource, toState, settings, name, recallid)
        .then(
          response => {
            if (response.data.fields !== undefined) {
              if (settings) {
                this.$router.push('/sources/new/' + encodeURIComponent(mySource) + '/' + toState)
              }
              this.form = response.data
            } else {
              this.$root.$bvToast.toast(
                'Done', {
                  title: response.data._comment ? response.data._comment : 'Your new source' + ' has been added.',
                  autoHideDelay: 5000,
                  variant: 'success',
                  toaster: 'b-toaster-content-right'
                }
              )
              this.$router.push('/sources/' + encodeURIComponent(response.data._id))
            }
          },
          error => {
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'Something went wrong during source configuration validation',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            if (!settings) {
              this.$router.push('/sources/new')
            }
          })
    },

    previousState () {
      if (this.form.previousButtonState !== undefined) {
        if (this.form.previousButtonState === -1) {
          this.$router.push('/sources/new/')
        } else {
          this.loadState(this.form.previousButtonState, this.settings, this.mySrcName)
        }
      }
    },

    submitSettings () {
      if (this.form.nextButtonState !== undefined) {
        if (this.form.nextButtonState === -1) {
          this.$router.push('/sources/new/')
        } else {
          this.loadState(this.form.nextButtonState, this.settings, this.mySrcName)
        }
      } else if (this.form.nextButtonLink !== undefined) {
        window.location = this.form.nextButtonLink
      }
    },

    updateSourceSettingsForm () {
      var state = parseInt(this.$route.params.state)

      SourceSpecsApi.getSourceSpecs()
        .then(
          response => (this.sourceSpecs = response.data),
          error => {
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'Something went wrong during source configuration',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/sources/new/')
          })

      this.loadState(state, null, null, this.$route.query.recall)
    }
  }
}
</script>
