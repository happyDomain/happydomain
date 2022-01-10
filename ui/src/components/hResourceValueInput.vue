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
  <b-form-row
    v-show="alwaysShow || edit || value != null"
  >
    <label v-if="specs.label" :for="'spec-' + index + '-' + specs.id" :title="specs.label" class="col-md-4 col-form-label text-truncate text-md-right text-primary">{{ specs.label }}</label>
    <label v-else :for="'spec-' + index + '-' + specs.id" :title="specs.label" class="col-md-4 col-form-label text-truncate text-md-right text-primary">{{ specs.id }}</label>
    <b-col md="8">
      <h-resource-value-input-raw v-model="val" :edit="edit" :index="index" :specs="specs" @focus="$emit('focus')" />
      <p v-if="specs.description" v-show="showDescription || specs.choices !== undefined" class="text-justify" style="line-height: 1.1">
        <small class="text-muted">{{ specs.description }}</small>
      </p>
    </b-col>
  </b-form-row>
</template>

<script>
export default {
  name: 'HResourceValueInput',

  components: {
    hResourceValueInputRaw: () => import('@/components/hResourceValueInputRaw')
  },

  props: {
    alwaysShow: {
      type: Boolean,
      default: false
    },
    edit: {
      type: Boolean,
      default: false
    },
    index: {
      type: Number,
      required: true
    },
    showDescription: {
      type: Boolean,
      default: true
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

  data: function () {
    return {
      show_description: false
    }
  },

  computed: {
    val: {
      get () {
        return this.value
      },
      set (val) {
        this.$emit('input', val)
      }
    }
  },

  methods: {
    saveChildrenValues () {}
  }
}
</script>
