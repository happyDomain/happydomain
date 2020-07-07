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
  <div v-if="!isLoading && (edit || tmp_values.length > 0)">
    <h4 v-if="specs.label" class="mt-1 text-primary">
      {{ specs.label }}
      <small v-if="specs.description" class="text-muted">{{ specs.description }}</small>
    </h4>
    <b-table hover striped :fields="fieldsNames" :items="tmp_values" sort-icon-left>
      <template v-slot:head(_actions)>
        <b-button size="sm" title="Add item" variant="outline-secondary" class="mx-1" @click="addRow()">
          <b-icon icon="plus" /> Add
        </b-button>
      </template>
      <template v-slot:cell()="row">
        <h-resource-value v-if="service_specs.fields" v-model="row.item[row.field.key]" :edit="row.item._edit" :index="row.index" :services="services" :specs="service_specs.fields[row.field.index]" :type="service_specs.fields[row.field.index].type" no-decorate @saveService="$emit('saveService', $event)" />
        <h-resource-value v-else v-model="row.item[row.field.key]" :edit="row.item._edit" :index="row.index" :services="services" :specs="specs" :type="row_type" no-decorate @saveService="$emit('saveService', $event)" />
      </template>
      <template v-slot:cell(_actions)="row">
        <b-button v-if="!row.item._edit" size="sm" title="Edit" variant="outline-primary" class="mx-1" @click="row.item._edit = !row.item._edit">
          <b-icon icon="pencil" />
        </b-button>
        <b-button v-else type="button" title="Save the modifications" size="sm" variant="success" class="mx-1" @click="saveRow(row)">
          <b-icon icon="check" />
        </b-button>
        <b-button v-if="!row.item._edit" type="button" title="Delete" size="sm" variant="outline-danger" class="mx-1" @click="deleteRow(row)">
          <b-icon icon="trash" />
        </b-button>
        <b-button v-else type="button" title="Cancel" size="sm" variant="danger" class="mx-1" @click="cancelEdit(row)">
          <b-icon icon="x-circle" />
        </b-button>
      </template>
    </b-table>
  </div>
</template>

<script>
import ServiceSpecsApi from '@/services/ServiceSpecsApi'

export default {
  name: 'HResourceValueTable',

  components: {
    hResourceValue: () => import('@/components/hResourceValue')
  },

  props: {
    edit: {
      type: Boolean,
      required: true
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
    value: {
      type: Array,
      default: null
    }
  },

  data: function () {
    return {
      row_type: '',
      service_specs: null,
      tmp_values: []
    }
  },

  computed: {
    isLoading () {
      return this.service_specs === null
    },
    fieldsNames () {
      var ret = []
      if (this.service_specs && this.service_specs.fields) {
        this.service_specs.fields.forEach(function (sspec, idx) {
          if (sspec.label) {
            ret.push({
              key: sspec.id,
              sortable: true,
              index: idx,
              label: sspec.label
            })
          } else {
            ret.push({
              key: sspec.id,
              sortable: true,
              index: idx
            })
          }
        })
      } else if (this.specs.label) {
        ret.push({ key: 'value', label: this.specs.label })
      } else {
        ret.push({ key: 'value', label: this.specs.id })
      }
      ret.push({ key: '_actions', label: '' })
      return ret
    }
  },

  watch: {
    service: function () {
      this.pullServiceSpecs()
    },
    value: function () {
      this.updateValues()
    }
  },

  created () {
    if (this.type !== undefined) {
      this.pullServiceSpecs()
    }
    if (this.value !== undefined) {
      this.updateValues()
    }
  },

  methods: {
    pullServiceSpecs () {
      this.row_type = this.type.substr(2)
      if (this.row_type === 'string') {
        this.service_specs = {}
      } else {
        ServiceSpecsApi.getServiceSpecs(this.row_type)
          .then(
            (response) => {
              this.service_specs = response.data
            }
          )
      }
    },

    addRow () {
      if ((this.value == null && this.tmp_values.length <= 1) || (this.value != null && this.tmp_values.length <= this.value.length)) {
        this.tmp_values.push({ _key: this.tmp_values.length, _edit: true })
      }
    },

    cancelEdit (row) {
      row.item._edit = false
      if (typeof this.value[row.item._key] === 'object') {
        Object.keys(this.value[row.item._key]).forEach(function (k) {
          row.item[k] = this.value[row.item._key][k]
        }, this)
      } else {
        row.item.value = this.value[row.item._key]
      }
    },

    deleteRow (row) {
      this.value.splice(row.item._key, 1)
      this.$emit('saveService')
    },

    saveRow (row) {
      if (this.service_specs && this.service_specs.fields) {
        var val = {}
        this.service_specs.fields.forEach(function (sspec, idx) {
          val[sspec.id] = row.item[sspec.id]
        }, this)

        if (this.value !== null && this.value[row.item._key] !== undefined) {
          this.value[row.item._key] = val
          this.$emit('input', this.value)
        } else if (this.value === null) {
          this.$emit('input', [val])
        } else {
          this.value.push(val)
          this.$emit('input', this.value)
        }
      } else if (this.value === null) {
        this.$emit('input', [row.item.value])
      } else {
        this.value[row.item._key] = row.item.value
      }

      this.$emit('saveService', function () {
        row.item._edit = false
      })
    },

    updateValues () {
      this.tmp_values = []

      if (this.value !== null) {
        this.value.forEach(function (v, k) {
          if (typeof v === 'object') {
            this.tmp_values.push(Object.assign({ _key: k, _edit: false }, v))
          } else {
            this.tmp_values.push({ _key: k, _edit: false, value: v })
          }
        }, this)
      }
    }
  }
}
</script>
