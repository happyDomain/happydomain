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
  <div class="d-flex flex-column">
    <div v-if="isLoading" class="mt-5 d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" :label="$t('common.spinning')" class="mr-3" /> {{ $t('wait.retrieving-setting') }}
    </div>
    <b-row v-else class="flex-grow-1">
      <b-col lg="4" md="5" class="bg-light">
        <div class="text-center mb-3">
          <img :src="'/api/provider_specs/' + $route.params.provider + '/icon.png'" :alt="providerSpecs_getAll[$route.params.provider].name" style="max-width: 100%; max-height: 10em">
        </div>
        <h3>
          {{ providerSpecs_getAll[$route.params.provider].name }}
        </h3>

        <p class="text-muted text-justify">
          {{ providerSpecs_getAll[$route.params.provider].description }}
        </p>

        <hr v-if="form.sideText">
        <p v-if="form.sideText" class="text-justify">
          {{ form.sideText }}
        </p>
      </b-col>

      <b-col lg="8" md="7">
        <b-form @submit.stop.prevent="nextState">
          <h-provider-state v-model="settings" class="mt-2 mb-2" :form="form" :provider-name="providerSpecs_getAll[$route.params.provider].name" :state="parseInt($route.params.state)" />
          <h-provider-state-buttons class="d-flex justify-content-end" :form="form" :next-is-working="nextIsWorking" :previous-is-working="previousIsWorking" @previous-state="previousState" />
        </b-form>
      </b-col>
    </b-row>
  </div>
</template>

<script>
import ProviderState from '@/mixins/providerState'
import { mapGetters } from 'vuex'

export default {

  components: {
    hProviderState: () => import('@/components/hProviderState'),
    hProviderStateButtons: () => import('@/components/hProviderStateButtons')
  },

  mixins: [ProviderState],

  data () {
    return {
      providerSpecsSelected: null
    }
  },

  computed: {
    isLoading () {
      return this.providerState_isLoading || this.providerSpecs_getAll == null
    },

    ...mapGetters('providerSpecs', ['providerSpecs_getAll'])
  },

  watch: {
    state: function (state) {
      if (state === -1) {
        this.$router.push('/providers/new/')
      } else if (state !== undefined && state !== this.$route.params.state) {
        this.$router.push('/providers/new/' + encodeURIComponent(this.providerSpecsSelected) + '/' + state)
      }
    }
  },

  created () {
    this.providerSpecsSelected = this.$route.params.provider
    this.state = parseInt(this.$route.params.state)
    this.resetSettings()
    this.updateSettingsForm()
  },

  methods: {
    reactOnSuccess (toState, newProvider) {
      if (newProvider) {
        this.$router.push('/providers/' + encodeURIComponent(newProvider._id) + '/domains')
      }
    }
  }
}
</script>
