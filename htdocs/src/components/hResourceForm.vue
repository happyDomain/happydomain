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
  <b-list-group-item v-if="!isLoading">
    <div class="text-right">
      <b-button v-if="!editService" type="button" size="sm" variant="outline-primary" class="mx-1" @click="toogleServiceEdit()">
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
    <h-form-data
      v-model="service.Service"
      :edit="editService"
      :fields="service_specs.fields"
    />
  </b-list-group-item>
</template>

<script>
import ServicesApi from '@/services/ServicesApi'

export default {
  name: 'HResourceForm',

  components: {
    hFormData: () => import('@/components/hFormData')
  },

  props: {
    service: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      editService: false,
      service_specs: null
    }
  },

  computed: {
    isLoading () {
      return this.service_specs == null
    }
  },

  watch: {
    service: function () {
      this.pullServiceSpecs()
    }
  },

  created () {
    if (this.service !== undefined) {
      this.pullServiceSpecs()
    }
  },

  methods: {
    pullServiceSpecs () {
      ServicesApi.getServiceSpecs(this.service._svctype)
        .then(
          (response) => {
            this.service_specs = response.data
          }
        )
    },

    toogleServiceEdit () {
      this.editService = !this.editService
    }
  }
}
</script>
