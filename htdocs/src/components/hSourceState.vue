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
  <h-custom-form v-if="val" :form="form" :value="val.Source" @focus="$emit('focus', $event)" @input="val.Source = $event; $emit('input', val)">
    <h-resource-value-simple-input
      v-if="state === 0"
      id="src-name"
      edit
      :index="0"
      :label="$t('source.name-your')"
      :description="$t('domains.give-explicit-name')"
      :placeholder="sourceName + ' account 1'"
      required
      :value="val._comment"
      @focus="$emit('focus', $event)"
      @input="val._comment = $event;$emit('input', val)"
    />
  </h-custom-form>
</template>

<script>
export default {
  name: 'HSourceState',

  components: {
    hCustomForm: () => import('@/components/hCustomForm'),
    hResourceValueSimpleInput: () => import('@/components/hResourceValueSimpleInput')
  },

  props: {
    form: {
      type: Object,
      required: true
    },
    sourceName: {
      type: String,
      required: true
    },
    state: {
      type: Number,
      default: 0
    },
    value: {
      type: Object,
      required: true
    }
  },

  data () {
    return {
      val: null
    }
  },

  watch: {
    value: function () {
      this.updateValues()
    }
  },

  mounted () {
    if (this.value) {
      this.updateValues()
    }
  },

  methods: {
    updateValues () {
      let nVal = {}
      if (this.value != null) {
        nVal = Object.assign({}, this.value)
      }
      if (nVal.Source === undefined) {
        nVal.Source = {}
      }
      if (nVal._comment === undefined) {
        nVal._comment = ''
      }
      this.val = nVal
    }
  }
}
</script>
