<!--
    Copyright or © or Copr. happyDNS (2020)

    contact@happydomain.org

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
  <b-modal
    id="modal-add-provider"
    scrollable
    size="lg"
    :title="$t('provider.new-form')"
    ok-title="Next >"
    :ok-disabled="!providerSpecsSelected"
    @ok="nextState"
  >
    <template v-if="state >= 0 && form" #modal-footer>
      <h-provider-state-buttons
        :form="form"
        submit-form="provider-state-form"
        :next-is-working="nextIsWorking"
        :previous-is-working="previousIsWorking"
        @previous-state="previousState"
      />
    </template>

    <div v-if="state < 0">
      <p>
        {{ $t('provider.select-provider') }}
      </p>
      <h-new-provider-selector v-model="providerSpecsSelected" />
    </div>

    <b-form v-else-if="providerSpecsSelected" id="provider-state-form" @submit.stop.prevent="nextState">
      <h-provider-state
        v-model="settings"
        class="mt-2 mb-2"
        :form="form"
        :provider-name="providerSpecs_getAll[providerSpecsSelected].name"
        :state="state"
      />
    </b-form>
  </b-modal>
</template>

<script>
import ProviderState from '@/mixins/providerState'
import { mapGetters } from 'vuex'

export default {
  name: 'HModalAddProvider',

  components: {
    hNewProviderSelector: () => import('@/components/hNewProviderSelector'),
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
    ...mapGetters('providerSpecs', ['providerSpecs_getAll'])
  },

  methods: {
    hide () {
      this.$bvModal.hide('modal-add-provider')
    },

    reactOnSuccess (toState, newProvider) {
      if (newProvider) {
        this.$emit('update-my-providers', newProvider)
        this.hide()
      }
    },

    show () {
      this.providerSpecsSelected = ''
      this.resetSettings()
      this.settings.redirect = window.location.pathname
      this.state = -1
      this.resetSettings()
      this.updateSettingsForm()
      this.$bvModal.show('modal-add-provider')
    }
  }
}
</script>
