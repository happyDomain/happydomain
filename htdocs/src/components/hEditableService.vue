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
  <h-resource-value v-model="service.Service" :edit="edit" :edit-toolbar="editToolbar" :services="services" :type="service._svctype" @deleteService="deleteService($event)" @saveService="saveService($event)" />
</template>

<script>
import ZoneApi from '@/services/ZoneApi'

export default {
  name: 'HEditableService',

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
    origin: {
      type: String,
      required: true
    },
    service: {
      type: Object,
      required: true
    },
    services: {
      type: Object,
      required: true
    },
    zoneId: {
      type: Number,
      required: true
    }
  },

  methods: {
    deleteService () {
      ZoneApi.deleteZoneService(this.origin, this.zoneId, this.service)
        .then(
          (response) => {
            this.$emit('updateMyServices', response.data)
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.occurs', { when: 'deleting the service' }),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    saveService (cbSuccess, cbFail) {
      if (this.service.Service === undefined) {
        this.deleteService()
      } else {
        ZoneApi.updateZoneService(this.origin, this.zoneId, this.service)
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
                  title: this.$t('errors.occurs', { when: 'updating the service' }),
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
    }
  }
}
</script>
