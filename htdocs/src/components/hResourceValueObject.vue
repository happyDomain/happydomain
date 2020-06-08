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
  <div v-if="service_specs && type" class="mb-2">
    <b-tabs v-if="services[type] && services[type].tabs" content-class="mt-3" fill>
      <b-tab v-for="(spec, index) in service_specs.fields" :key="index" :active="index === 0">
        <template v-slot:title>
          {{ spec | hLabel }}
          <b-badge v-if="spec.type.substr(0,2) === '[]'" variant="light" pill>
            {{ value[spec.id].length }}
          </b-badge>
          <b-badge v-if="spec.type.substr(0,3) === 'map'" variant="light" pill>
            {{ Object.keys(value[spec.id]).length }}
          </b-badge>
        </template>
        <h-resource-value
          v-if="value[spec.id]"
          v-model="value[spec.id]"
          :edit="editChildren"
          :edit-toolbar="editToolbar"
          :services="services"
          :specs="spec"
          :type="spec.type"
        />
        <b-button v-else>
          Create {{ spec.id }}
        </b-button>
      </b-tab>
    </b-tabs>
    <div v-else>
      <div v-if="editToolbar" class="text-right mb-2">
        <b-button v-if="!serviceEdit" type="button" size="sm" variant="outline-primary" class="mx-1" @click="toogleServiceEdit()">
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
      <h-resource-value
        v-for="(spec, index) in service_specs.fields"
        :key="index"
        v-model="value[spec.id]"
        :edit="editChildren"
        :services="services"
        :specs="spec"
        :type="spec.type"
      />
    </div>
  </div>
</template>

<script>
import ServicesApi from '@/services/ServicesApi'

export default {
  name: 'HResourceValueObject',

  components: {
    hResourceValue: () => import('@/components/hResourceValue')
  },

  props: {
    edit: {
      type: Boolean,
      default: false
    },
    editToolbar: {
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

  data () {
    return {
      serviceEdit: false,
      service_specs: null
    }
  },

  computed: {
    editChildren () {
      return this.edit || this.serviceEdit
    }
  },

  watch: {
    type: function () {
      this.pullServiceSpecs()
    }
  },

  created () {
    if (this.type !== undefined) {
      this.pullServiceSpecs()
    }
  },

  methods: {
    pullServiceSpecs () {
      ServicesApi.getServiceSpecs(this.type)
        .then(
          (response) => {
            this.service_specs = response.data
          }
        )
    },

    toogleServiceEdit () {
      this.serviceEdit = !this.serviceEdit
    }
  }
}
</script>
