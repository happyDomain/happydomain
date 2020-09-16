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
  <b-form v-if="!isLoading" class="mt-2 mb-5" @submit.stop.prevent="nextState">
    <b-button v-if="!edit" class="float-right" type="button" size="sm" variant="outline-primary" @click="editSource">
      <b-icon icon="pencil" />
      {{ $t('common.edit') }}
    </b-button>

    <h-resource-value-simple-input
      v-if="!edit"
      id="src-name"
      v-model="mySource._comment"
      always-show
      :index="0"
      :label="$t('source.source-name')"
      :description="edit?'Give an explicit name in order to easily find this service.':''"
      :placeholder="sources[sourceSpecsSelected].name + ' 1'"
      required
    />
    <h-source-state v-else-if="form" v-model="settings" class="mt-2 mb-2" :form="form" :source-name="sources[sourceSpecsSelected].name" :state="state" />

    <hr>

    <h-fields v-if="!edit" v-model="mySource.Source" :fields="sourceSpecs.fields" />

    <h-source-state-buttons v-else-if="form" class="d-flex justify-content-end" edit :form="form" :next-is-working="nextIsWorking" :previous-is-working="previousIsWorking" @previousState="previousState" />
  </b-form>
</template>

<script>
import SourceState from '@/mixins/sourceState'

export default {

  components: {
    hFields: () => import('@/components/hFields'),
    hResourceValueSimpleInput: () => import('@/components/hResourceValueSimpleInput'),
    hSourceState: () => import('@/components/hSourceState'),
    hSourceStateButtons: () => import('@/components/hSourceStateButtons')
  },

  mixins: [SourceState],

  props: {
    parentLoading: {
      type: Boolean,
      required: true
    },
    mySource: {
      type: Object,
      default: null
    },
    sources: {
      type: Object,
      default: null
    },
    sourceSpecs: {
      type: Object,
      default: null
    },
    sourceSpecsSelected: {
      type: String,
      default: null
    }
  },

  data: function () {
    return {
      edit: false
    }
  },

  computed: {
    isLoading () {
      return this.parentLoading
    }
  },

  watch: {
    state: function (state) {
      if (state === -1) {
        this.edit = false
      }
    }
  },

  methods: {
    editSource () {
      this.settings = Object.assign({}, this.mySource)
      this.loadState(0)
      this.edit = true
    },

    reactOnSuccess (toState, newSource) {
      if (newSource) {
        this.$emit('updateMySource', newSource)
        this.edit = false
      }
    }
  }
}
</script>
