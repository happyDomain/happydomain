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
  <b-modal id="modal-addSvc" :size="step === 2 ? 'lg' : ''" scrollable @ok="handleModalSvcOk">
    <template #modal-title>
      <i18n path="service.form-new">
        <span class="text-monospace">{{ realSubDomain }}</span>
      </i18n>
    </template>
    <template #modal-footer="{ cancel }">
      <h-help
        v-if="step === 2"
        :href="helpHref"
        variant="info"
      />
      <b-button
        v-if="update"
        :disabled="deleteServiceInProgress || !svcData || svcData._svctype === 'abstract.Origin'"
        variant="danger"
        @click="deleteService(svcData)"
      >
        <b-spinner v-if="deleteServiceInProgress" label="Spinning" small />
        {{ $t('service.delete') }}
      </b-button>
      <b-button variant="secondary" @click="cancel()">
        {{ $t('common.cancel') }}
      </b-button>
      <b-button
        v-if="step === 2 && update"
        :disabled="addServiceInProgress"
        form="addSvcForm"
        type="submit"
        variant="success"
      >
        <b-spinner v-if="addServiceInProgress" label="Spinning" small />
        {{ $t('service.update') }}
      </b-button>
      <b-button
        v-else-if="step === 2"
        form="addSvcForm"
        type="submit"
        variant="primary"
      >
        {{ $t('service.add') }}
      </b-button>
      <b-button
        v-else
        :disabled="(step === 0 && !validateNewSubdomain()) || (step === 1 && !svcSelected)"
        form="addSvcForm"
        type="submit"
        variant="primary"
      >
        {{ $t('common.continue') }}
      </b-button>
    </template>
    <form id="addSvcForm" @submit.stop.prevent="handleModalSvcOk">
      <p v-if="step === 0">
        <i18n path="domains.form-new-subdomain">
          <span class="text-monospace">{{ domain.domain }}</span>
        </i18n>
        <b-input-group :append="newDomainAppend">
          <b-input
            v-model="dn"
            autofocus
            class="text-monospace"
            :placeholder="$t('domains.placeholder-new-sub')"
            :state="newDomainState"
            @update="validateNewSubdomain"
          />
        </b-input-group>
      </p>
      <h-family-tabs
        v-else-if="step === 1"
        v-model="svcSelected"
        class="mb-2"
        content-class="mt-3"
        :domain="domain"
        :dn="dn"
        :my-services="myServices"
        :services="services"
      />
      <div v-else-if="step === 2">
        <h-custom-form
          v-if="form"
          ref="addModalResources"
          v-model="svcData.Service"
          :form="form"
          :services="services"
          :type="svcSelected"
        />
        <b-spinner v-else label="Spinning" />
      </div>
    </form>
  </b-modal>
</template>

<script>
import ServicesApi from '@/api/services'
import CustomForm from '@/mixins/customForm'
import ValidateDomain from '@/mixins/validateDomain'
import ZoneApi from '@/api/zones'

export default {
  name: 'HModalAddService',

  components: {
    hCustomForm: () => import('@/components/hCustomForm'),
    hFamilyTabs: () => import('@/components/hFamilyTabs'),
    hHelp: () => import('@/components/hHelp')
  },

  mixins: [CustomForm, ValidateDomain],

  props: {
    domain: {
      type: Object,
      required: true
    },
    myServices: {
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

  data: function () {
    return {
      addServiceInProgress: false,
      deleteServiceInProgress: false,
      dn: '',
      newDomainState: null,
      step: 0,
      svcData: {},
      svcSelected: null,
      update: false
    }
  },

  computed: {
    endsWithOrigin () {
      return this.dn.length > this.domain.domain.length && (this.dn.substring(this.dn.length - this.domain.domain.length) === this.domain.domain || this.dn.substring(this.dn.length - this.domain.domain.length + 1) === this.domain.domain.substring(0, this.domain.domain.length - 1))
    },

    helpHref () {
      const svcPart = this.svcData._svctype.toLowerCase().split('.')
      if (svcPart.length === 2) {
        if (svcPart[0] === 'svcs') {
          return 'records/' + svcPart[1].toUpperCase()
        } else if (svcPart[0] === 'abstract') {
          return 'services/' + svcPart[1]
        } else if (svcPart[0] === 'provider') {
          return 'services/providers/' + svcPart[1]
        }
      }
      return svcPart[svcPart.length - 1]
    },

    newDomainAppend () {
      if (this.endsWithOrigin) {
        return null
      } else if (this.dn.length > 0) {
        return '.' + this.domain.domain
      } else {
        return this.domain.domain
      }
    },

    realSubDomain () {
      return this.dn + (this.newDomainAppend ? this.newDomainAppend : '')
    }
  },

  methods: {
    deleteService (service) {
      this.deleteServiceInProgress = true
      ZoneApi.deleteZoneService(this.domain.domain, this.zoneId, service)
        .then(
          (response) => {
            this.$bvModal.hide('modal-addSvc')
            this.deleteServiceInProgress = false
            this.$emit('update-my-services', response.data)
          },
          (error) => {
            this.deleteServiceInProgress = false
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

    getFormSettings (state, settings, recallid) {
      return ServicesApi.getFormSettings(this.domain.domain, this.zoneId, this.dn, this.svcSelected, state, settings, recallid)
    },

    handleModalSvcOk (bvModalEvt) {
      bvModalEvt.preventDefault()

      if (this.step === 0 && this.dn !== '') {
        if (this.validateNewSubdomain()) {
          this.step = 1
          if (this.endsWithOrigin) {
            if (this.dn.substring(this.dn.length - this.domain.domain.length) === this.domain.domain) {
              this.dn = this.dn.substring(0, this.dn.length - this.domain.domain.length - 1)
            } else {
              this.dn = this.dn.substring(0, this.dn.length - this.domain.domain.length)
            }
          }
        }
      } else if (this.step === 1 && this.svcSelected !== null) {
        this.step = 2
        this.svcData = { Service: {}, _svctype: this.svcSelected }
        this.resetSettings()
        this.updateSettingsForm()
      } else if (this.step === 2 && this.svcSelected !== null) {
        this.$refs.addModalResources.saveChildrenValues()

        let func = null
        if (this.update) {
          func = ZoneApi.updateZoneService
        } else {
          func = ZoneApi.addZoneService
        }

        func(this.domain.domain, this.zoneId, this.dn, this.svcData)
          .then(
            (response) => {
              this.$emit('update-my-services', response.data)
              this.$nextTick(() => {
                this.$bvModal.hide('modal-addSvc')
              })
            },
            (error) => {
              this.$root.$bvToast.toast(
                error.response.data.errmsg, {
                  title: 'Unable to add the new service',
                  autoHideDelay: 5000,
                  variant: 'danger',
                  toaster: 'b-toaster-content-right'
                }
              )
            }
          )
      }
    },

    show (dn, data) {
      this.addServiceInProgress = false
      this.deleteServiceInProgress = false
      this.newDomainState = null

      if (dn !== undefined) {
        this.step = 1
        this.dn = dn
      } else {
        this.step = 0
        this.dn = ''
      }

      if (data !== undefined) {
        this.step = 2
        this.svcSelected = data._svctype
        this.svcData = data
        this.update = true
        this.updateSettingsForm()
      } else {
        this.svcSelected = null
        this.svcData = { Service: {} }
        this.update = false
      }

      this.$bvModal.show('modal-addSvc')
    },

    validateNewSubdomain () {
      this.newDomainState = this.validateDomain(
        this.dn,
        // true, except when it ends with the origin name
        !(this.dn.length > this.domain.domain.length && this.dn.substring(this.dn.length - this.domain.domain.length) === this.domain.domain)
      )
      return this.newDomainState
    }
  }
}
</script>
