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
  <div>
    <p v-if="form.beforeText" class="lead text-indent">
      {{ form.beforeText }}
    </p>
    <p v-else>
      {{ $t('domains.please-fill-fields') }}
    </p>

    <h-resource-value-simple-input
      v-if="state === 0"
      id="src-name"
      edit
      :index="0"
      :label="$t('domains.name-your-source')"
      :description="$t('domains.give-explicit-name')"
      :placeholder="sourceName + ' account 1'"
      required
      :value="val._comment"
      @input="val._comment = $event;$emit('input', val)"
    />

    <h-fields
      v-if="form.fields && val.Source"
      edit
      :fields="form.fields"
      :value="val.Source"
      @input="val.Source = $event;$emit('input', val)"
    />

    <p v-if="form.afterText">
      {{ form.afterText }}
    </p>
  </div>
</template>

<script>
export default {
  name: 'HSourceState',

  components: {
    hFields: () => import('@/components/hFields'),
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
      val: {}
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
      if (this.value != null) {
        this.val = Object.assign({}, this.value)
      } else {
        this.val = {}
      }
      if (this.val.Source === undefined) {
        this.val.Source = {}
      }
      if (this.val._comment === undefined) {
        this.val._comment = ''
      }
    }
  }
}
</script>
