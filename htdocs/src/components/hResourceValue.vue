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
  <component :is="itemComponent" :value="value" @input="$emit('input', $event)" :edit="edit" :edit-toolbar="editToolbar" :index="index" :services="services" :specs="specs" :type="type" @saveService="$emit('saveService', $event)" />
</template>

<script>
import HResourceValueTable from '@/components/hResourceValueTable'
import HResourceValueMap from '@/components/hResourceValueMap'
import HResourceValueObject from '@/components/hResourceValueObject'
import HResourceValueInput from '@/components/hResourceValueInput'
import HResourceValueInputRaw from '@/components/hResourceValueInputRaw'

export default {
  name: 'HResourceValue',

  props: {
    edit: {
      type: Boolean,
      default: false
    },
    editToolbar: {
      type: Boolean,
      default: false
    },
    index: {
      type: Number,
      default: NaN
    },
    noDecorate: {
      type: Boolean,
      default: false
    },
    services: {
      type: Object,
      required: true
    },
    specs: {
      type: Object,
      default: null
    },
    type: {
      type: String,
      required: true
    },
    // eslint-disable-next-line
    value: {
      required: true
    }
  },

  data () {
    return {
      itemComponent: ''
    }
  },

  watch: {
    type: function () {
      this.findComponent()
    }
  },

  mounted () {
    if (this.type) {
      this.findComponent()
    }
  },

  methods: {
    findComponent () {
      if (Array.isArray(this.value)) {
        this.itemComponent = HResourceValueTable
      } else if (this.type.substr(0, 3) === 'map') {
        this.itemComponent = HResourceValueMap
      } else if (typeof this.value === 'object' || this.type.substr(0, 6) === '*svcs.') {
        this.itemComponent = HResourceValueObject
      } else if (this.noDecorate) {
        this.itemComponent = HResourceValueInputRaw
      } else {
        this.itemComponent = HResourceValueInput
      }
    }
  }
}
</script>
