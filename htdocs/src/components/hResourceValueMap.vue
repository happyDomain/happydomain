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
  <div ng-if="!isLoading()">
    <div class="text-right">
      <b-button v-if="!edit" type="button" size="sm" variant="outline-primary" class="mx-1" @click="toogleServiceEdit()">
        <b-icon icon="pencil" />
        Edit
      </b-button>
      <b-button v-else type="button" size="sm" variant="primary" class="mx-1" @click="submitService(index, idx)">
        <b-icon icon="check" />
        Save those modifications
      </b-button>
      <b-button type="button" size="sm" variant="outline-danger" class="mx-1">
        <b-icon icon="trash" />
        Delete
      </b-button>
    </div>
    <div v-for="(val,key) in value" :key="key">
      <h3>{{ key }}</h3>
      <h-resource-value
        v-model="value[key]"
        :edit="edit"
        :services="services"
        :specs="service_specs"
        :type="main_type"
      />
      <hr>
    </div>
  </div>
</template>

<script>
import ServicesApi from '@/services/ServicesApi'

export default {
  name: 'HResourceValueMap',

  components: {
    hResourceValue: () => import('@/components/hResourceValue')
  },

  props: {
    edit: {
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
    value: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      key_type: '',
      main_type: '',
      service_specs: null
    }
  },

  computed: {
    isLoading () {
      return this.service_specs == null
    },
    fieldsNames () {
      var ret = []
      this.service_specs.fields.forEach(function (sspec) {
        ret.push({
          key: sspec.id,
          sortable: true
        })
      })
      return ret
    }
  },

  watch: {
    type: function () {
      this.key_type = this.type.substr(this.type.indexOf('[') + 1, this.type.indexOf(']') - this.type.indexOf('['))
      this.main_type = this.type.substr(this.type.indexOf(']') + 1)
      this.pullServiceSpecs()
    }
  },

  created () {
    if (this.type !== undefined) {
      this.key_type = this.type.substr(this.type.indexOf('[') + 1, this.type.indexOf(']') - this.type.indexOf('['))
      this.main_type = this.type.substr(this.type.indexOf(']') + 1)
      this.pullServiceSpecs()
    }
  },

  methods: {
    pullServiceSpecs () {
      ServicesApi.getServiceSpecs(this.main_type)
        .then(
          (response) => {
            this.service_specs = response.data
          }
        )
    },

    toogleServiceEdit () {
      this.edit = !this.edit
    }
  }
}
</script>
