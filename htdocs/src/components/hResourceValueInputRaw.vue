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
  <b-input-group size="sm" :append="unit">
    <b-form-select
      v-if="edit && specs.choices !== undefined"
      :id="'spec-' + index + '-' + specs.id"
      v-model="val"
      :required="specs.required !== undefined && specs.required"
      :options="specs.choices"
    />
    <b-form-input
      v-else
      :id="'spec-' + index + '-' + specs.id"
      v-model="val"
      class="font-weight-bold"
      lazy
      :required="specs.required !== undefined && specs.required"
      :placeholder="specs.placeholder"
      :plaintext="!edit"
    />
  </b-input-group>
</template>

<script>
export default {
  name: 'HResourceValueInputRaw',

  props: {
    edit: {
      type: Boolean,
      default: false
    },
    index: {
      type: Number,
      required: true
    },
    specs: {
      type: Object,
      default: null
    },
    // eslint-disable-next-line
    value: {
      required: true
    }
  },

  computed: {
    unit () {
      if (this.specs.type === 'time.Duration') {
        return 's'
      } else {
        return null
      }
    },
    val: {
      get () {
        if (this.specs.type === 'time.Duration') {
          return this.value / 1000000000
        } else if (this.specs.type === '[]uint8') {
          const raw = atob(this.value)
          let result = ''
          for (let i = 0; i < raw.length; i++) {
            const hex = raw.charCodeAt(i).toString(16)
            result += (hex.length === 2 ? hex : '0' + hex)
          }
          return result.toUpperCase()
        } else {
          return this.value
        }
      },
      set (value) {
        if (this.specs.type === 'time.Duration') {
          this.$emit('input', value * 1000000000)
        } else if (this.specs.type === 'int' || this.specs.type === 'int8' || this.specs.type === 'int16' || this.specs.type === 'int32' || this.specs.type === 'int64' || this.specs.type === 'uint' || this.specs.type === 'uint8' || this.specs.type === 'uint16' || this.specs.type === 'uint32' || this.specs.type === 'uint64') {
          this.$emit('input', parseInt(value, 10))
        } else if (this.specs.type === '[]uint8') {
          let res = ''
          if (value.length % 2) {
            res = ('0' + value).match(/\w{2}/g).map(function (a) { return String.fromCharCode(parseInt(a, 16)) }).join('')
          } else {
            res = value.match(/\w{2}/g).map(function (a) { return String.fromCharCode(parseInt(a, 16)) }).join('')
          }
          this.$emit('input', btoa(res))
        } else {
          this.$emit('input', value)
        }
      }
    }
  }
}
</script>
