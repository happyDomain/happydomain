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
    <h-resource-value
      v-for="(specs, index) in fields"
      v-show="edit || !specs.secret"
      ref="child"
      :key="index"
      :edit="edit"
      :index="index"
      :services="services"
      :specs="specs"
      :type="specs.type"
      :value="val[specs.id]"
      @input="val[specs.id] = $event;$emit('input', val)"
    />
  </div>
</template>

<script>
export default {
  name: 'HFields',

  components: {
    hResourceValue: () => import('@/components/hResourceValue')
  },

  props: {
    edit: {
      type: Boolean,
      default: false
    },
    fields: {
      type: Array,
      required: true
    },
    services: {
      type: Object,
      default: () => {}
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
    fields: function () {
      this.fieldsUpdated()
    },
    value: function () {
      this.updateValues()
    }
  },

  created () {
    if (this.value !== undefined) {
      this.updateValues()
    }
  },

  methods: {
    fieldsUpdated () {
      for (const i in this.fields) {
        if (this.value[this.fields[i].id] === undefined && this.fields[i].default) {
          this.val[this.fields[i].id] = this.fields[i].default
        }
      }
    },

    saveChildrenValues () {
      this.$refs.child.forEach(function (row) {
        row.saveChildrenValues()
      }, this)
    },

    updateValues () {
      if (this.value != null) {
        this.val = Object.assign({}, this.value)
      } else {
        this.val = {}
      }

      if (this.fields !== undefined) {
        this.fieldsUpdated()
      }
    }
  }

}
</script>
