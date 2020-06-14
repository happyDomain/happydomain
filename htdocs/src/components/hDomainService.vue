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
  <b-list-group v-if="!isLoading">
    <b-list-group-item button @click="toogleShowDetails()">
      <strong :title="services[service._svctype].description">{{ services[service._svctype].name }}</strong> <span v-if="service._comment" class="text-muted">{{ service._comment }}</span>
      <span v-if="services[service._svctype].comment" class="text-muted">{{ services[service._svctype].comment }}</span>
      <b-badge v-for="(categorie, idcat) in services[service._svctype].categories" :key="idcat" variant="gray" class="float-right ml-1">
        {{ categorie }}
      </b-badge>
      <b-badge v-if="service._tmp_hint_nb && service._tmp_hint_nb > 1" variant="danger" class="float-right ml-1">
        {{ service._tmp_hint_nb }}
      </b-badge>
    </b-list-group-item>
    <b-list-group-item v-if="showDetails">
      <h-resource-value v-model="service.Service" edit-toolbar :services="services" :type="service._svctype" @deleteService="deleteService(service, $event)" @saveService="saveService(service, $event)" />
    </b-list-group-item>
  </b-list-group>
</template>

<script>
import ServiceSpecsApi from '@/services/ServiceSpecsApi'
import ZoneApi from '@/services/ZoneApi'

export default {
  name: 'HDomainService',

  components: {
    hResourceValue: () => import('@/components/hResourceValue')
  },

  props: {
    origin: {
      type: String,
      required: true
    },
    service: {
      type: Object,
      required: true
    },
    zoneMeta: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      showDetails: false,
      services: null
    }
  },

  computed: {
    isLoading () {
      return this.services == null
    }
  },

  created () {
    ServiceSpecsApi.getServiceSpecs()
      .then(
        (response) => (this.services = response.data)
      )
  },

  methods: {
    deleteService (service) {
      this.showDetails = false
      ZoneApi.deleteZoneService(this.origin, this.zoneMeta.id, service)
        .then(
          (response) => {
            this.$emit('updateMyServices', response.data)
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when deleting the service!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    saveService (service, cbSuccess, cbFail) {
      if (service.Service === undefined) {
        this.deleteService(service)
      } else {
        ZoneApi.updateZoneService(this.origin, this.zoneMeta.id, service)
          .then(
            (response) => {
              this.$emit('updateMyServices', response.data)
              if (cbSuccess != null) {
                cbSuccess()
              }
            },
            (error) => {
              this.$bvToast.toast(
                error.response.data.errmsg, {
                  title: 'An error occurs when updating the service!',
                  autoHideDelay: 5000,
                  variant: 'danger',
                  toaster: 'b-toaster-content-right'
                }
              )
              if (cbFail != null) {
                cbFail(error)
              }
            }
          )
      }
    },

    toogleShowDetails () {
      this.showDetails = !this.showDetails
    }
  }
}
</script>
