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
  <b-modal id="modal-add-source" scrollable size="lg" title="New Source Form" ok-title="Next >" :ok-disabled="!sourceSpecsSelected" @ok="nextState">
    <template v-if="state >= 0 && form" v-slot:modal-footer>
      <h-source-state-buttons :form="form" submit-form="source-state-form" :next-is-working="nextIsWorking" :previous-is-working="previousIsWorking" @previousState="previousState" />
    </template>

    <div v-if="state < 0">
      <p>
        {{ $t('source.select-provider') }}
      </p>
      <h-new-source-selector v-model="sourceSpecsSelected" />
    </div>

    <b-form v-else-if="sourceSpecsSelected" id="source-state-form" @submit.stop.prevent="nextState">
      <h-source-state v-model="settings" class="mt-2 mb-2" :form="form" :source-name="sourceSpecs[sourceSpecsSelected].name" :state="state" />
    </b-form>
  </b-modal>
</template>

<script>
import SourceSpecs from '@/mixins/sourceSpecs'
import SourceState from '@/mixins/sourceState'

export default {
  name: 'HModalAddSource',

  components: {
    hNewSourceSelector: () => import('@/components/hNewSourceSelector'),
    hSourceState: () => import('@/components/hSourceState'),
    hSourceStateButtons: () => import('@/components/hSourceStateButtons')
  },

  mixins: [SourceSpecs, SourceState],

  data () {
    return {
      sourceSpecsSelected: null
    }
  },

  methods: {
    hide () {
      this.$bvModal.hide('modal-add-source')
    },

    reactOnSuccess (toState, newSource) {
      if (newSource) {
        this.$emit('updateMySources', newSource)
        this.hide()
      }
    },

    show () {
      this.sourceSpecsSelected = ''
      this.resetSettings()
      this.settings.redirect = window.location.pathname
      this.state = -1
      this.resetSettings()
      this.updateSettingsForm()
      this.$bvModal.show('modal-add-source')
    }
  }
}
</script>
