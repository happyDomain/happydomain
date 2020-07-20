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
        <b-form @submit.stop.prevent="nextState">
          <h-source-state v-model="settings" class="mt-2 mb-2" :form="form" :source-name="sourceSpecs[$route.params.provider].name" :state="parseInt($route.params.state)" />
          <h-source-state-buttons class="d-flex justify-content-end" :form="form" :next-is-working="nextIsWorking" :previous-is-working="previousIsWorking" @previousState="previousState" />
        </b-form>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import SourceSpecs from '@/mixins/sourceSpecs'
import SourceState from '@/mixins/sourceState'

export default {

  components: {
    hSourceState: () => import('@/components/hSourceState'),
    hSourceStateButtons: () => import('@/components/hSourceStateButtons')
  },

  mixins: [SourceSpecs, SourceState],

  data () {
    return {
      sourceSpecsSelected: null
    }
  },

  watch: {
    state: function (state) {
      if (state === -1) {
        this.$router.push('/sources/new/')
      } else if (state !== undefined && state !== this.$route.params.state) {
        this.$router.push('/sources/new/' + encodeURIComponent(this.sourceSpecsSelected) + '/' + state)
      }
    }
  },

  created () {
    this.sourceSpecsSelected = this.$route.params.provider
    this.state = parseInt(this.$route.params.state)
    this.updateSourceSpecs()
  },

  methods: {
    reactOnSuccess (toState, newSource) {
      if (newSource) {
        this.$router.push('/sources/' + encodeURIComponent(newSource._id) + '/domains')
      }
    }
  }
}
</script>
