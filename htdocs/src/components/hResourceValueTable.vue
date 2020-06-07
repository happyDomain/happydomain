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
  <div v-if="!isLoading">
    <b-table hover striped :fields="fieldsNames" :items="value" sort-icon-left>
      <template v-slot:head(_actions)>
        <b-button size="sm" title="Add item" variant="outline-secondary" class="mx-1" @click="addRow()">
          <b-icon icon="plus" /> Add
        </b-button>
      </template>
      <template v-slot:cell()="row">
        <h-resource-value v-if="service_specs.fields" v-model="row.item[row.field.key]" :edit="edit_row.indexOf(row.index) >= 0" :index="row.index" :services="services" :specs="service_specs.fields[row.field.index]" :type="service_specs.fields[row.field.index].type" no-decorate />
        <h-resource-value v-else v-model="row.item" :edit="edit_row.indexOf(row.index) >= 0" :index="row.index" :services="services" :specs="specs" :type="row_type" no-decorate />
      </template>
      <template v-slot:cell(_actions)="row">
        <b-button v-if="edit_row.indexOf(row.index) < 0" size="sm" title="Edit" variant="outline-primary" class="mx-1" @click="editService(row)">
          <b-icon icon="pencil" />
        </b-button>
        <b-button v-else type="button" title="Save the modifications" size="sm" variant="primary" class="mx-1" @click="submitService(row.item)">
          <b-icon icon="check" />
        </b-button>
        <b-button type="button" title="Delete" size="sm" variant="outline-danger" class="mx-1" @click="deleteService(row.item)">
          <b-icon icon="trash" />
        </b-button>
      </template>
    </b-table>
  </div>
</template>

<script>
import ServicesApi from '@/services/ServicesApi'

export default {
  name: 'HResourceValueTable',

  components: {
    hResourceValue: () => import('@/components/hResourceValue')
  },

  props: {
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
      required: true
    }
  },

  data: function () {
    return {
      edit_row: [],
      row_type: '',
      service_specs: null
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
          ret.push({
            key: sspec.id,
            sortable: true,
            index: idx
          })
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
      this.row_type = this.type.substr(2)
      this.pullServiceSpecs()
    }
  },

  created () {
    if (this.type !== undefined) {
      this.row_type = this.type.substr(2)
      this.pullServiceSpecs()
    }
  },

  methods: {
    pullServiceSpecs () {
      if (this.row_type === 'string') {
        this.service_specs = {}
      } else {
        ServicesApi.getServiceSpecs(this.row_type)
          .then(
            (response) => {
              this.service_specs = response.data
            }
          )
      }
    },

    editService (row) {
      if (this.edit_row.indexOf(row.index) >= 0) {
        this.edit_row = this.edit_row.splice(this.edit_row.indexOf(row.index), 1)
      } else {
        this.edit_row.push(row.index)
      }
    }
  }
}
</script>
