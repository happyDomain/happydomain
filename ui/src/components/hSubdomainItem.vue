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
    <div v-if="isCNAME()">
      <h2 :id="dn" class="sticky-top" style="background: white; text-indent:-1em;padding-left:1em;overflow-x:hidden;text-overflow:ellipsis;z-index: 14">
        <span style="white-space: nowrap">
          <b-icon icon="link" />
          <span class="text-monospace" :title="dn | fqdn(origin)">{{ dn | fqdn(origin) }}</span>
        </span>
        <span style="white-space: nowrap">
          <b-icon icon="arrow-right" />
          <span class="text-monospace">{{ zoneServices[0].Service.Target }}</span>
        </span>
        <b-button type="button" variant="primary" size="sm" class="ml-2" @click="$emit('add-new-service', dn)">
          <b-icon icon="plus" />
          {{ $t('service.add') }}
        </b-button>
        <b-button type="button" variant="outline-info" size="sm" class="ml-2" @click="$emit('show-service-window', zoneServices[0])">
          <b-icon icon="pencil" />
          {{ $t('domains.edit-target') }}
        </b-button>
        <b-button type="button" variant="outline-danger" size="sm" class="ml-2" @click="deleteCNAME()">
          <b-icon icon="x-circle" />
          {{ $t('domains.drop-alias') }}
        </b-button>
      </h2>
    </div>
    <div v-else>
      <h2 :id="dn?dn:'@'" class="sticky-top" style="background: white; text-indent:-1em;padding-left:1em;overflow-x:hidden;text-overflow:ellipsis;z-index: 14">
        <span style="white-space: nowrap">
          <b-icon :icon="showResources?'chevron-down':'chevron-right'" @click="toogleShowResources()" />
          <span class="text-monospace" :title="dn | fqdn(origin)" @click="toogleShowResources()">{{ dn | fqdn(origin) }}</span>
        </span>
        <b-badge v-if="aliases.length > 0" v-b-popover.hover.focus="{ customClass: 'text-monospace', html: true, content: aliasPopoverCnt(dn) }" class="ml-2" style="text-indent:0;">
          + {{ $tc('domains.n-aliases', aliases.length) }}
        </b-badge>
        <b-button v-if="displayFormat !== 'grid'" type="button" variant="primary" size="sm" class="ml-2" @click="$emit('add-new-service', dn)">
          <b-icon icon="plus" />
          {{ $t('domains.add-a-service') }}
        </b-button>
        <b-button type="button" variant="outline-primary" size="sm" class="ml-2" @click="$emit('add-new-alias', dn)">
          <b-icon icon="link" />
          {{ $t('domains.add-an-alias') }}
        </b-button>
        <b-button v-if="!showSubdomainsList && !dn" type="button" variant="outline-secondary" size="sm" class="ml-2" @click="$emit('add-subdomain')">
          <b-icon icon="server" />
          {{ $t('domains.add-a-subdomain') }}
        </b-button>
      </h2>
      <div v-show="showResources" :class="showResources && displayFormat === 'grid' ? 'd-flex justify-content-around flex-wrap' : ''">
        <h-domain-service v-for="(svc, idx) in zoneServices" :key="idx" :display-format="displayFormat" :origin="origin" :service="svc" :services="services" :zone-id="zoneId" @show-service-window="$emit('show-service-window', $event)" @update-my-services="$emit('update-my-services', $event)" />
        <h-domain-service v-if="displayFormat === 'grid'" :display-format="displayFormat" :origin="origin" :services="services" :zone-id="zoneId" @show-service-window="$emit('add-new-service', dn)" />
      </div>
    </div>
  </div>
</template>

<script>
import ZoneApi from '@/api/zones'

export default {
  name: 'HSubdomainItem',

  components: {
    hDomainService: () => import('@/components/hDomainService')
  },

  props: {
    aliases: {
      type: Array,
      required: true
    },
    displayFormat: {
      type: String,
      default: 'grid'
    },
    dn: {
      type: String,
      required: true
    },
    origin: {
      type: String,
      required: true
    },
    showSubdomainsList: {
      type: Boolean,
      default: false
    },
    services: {
      type: Object,
      required: true
    },
    zoneId: {
      type: Number,
      required: true
    },
    zoneServices: {
      type: Array,
      required: true
    }
  },

  data: function () {
    return {
      showResources: true
    }
  },

  methods: {
    toogleShowResources () {
      this.showResources = !this.showResources
    },

    isCNAME () {
      return this.zoneServices.length === 1 && this.zoneServices[0]._svctype === 'svcs.CNAME'
    },

    aliasPopoverCnt () {
      return this.aliases.map(function (alias) {
        if (alias[-1] !== '.') {
          return '<a href="#' + this.escapeHTML(alias) + '">' + this.escapeHTML(alias) + '</a>'
        } else {
          return this.escapeHTML(alias)
        }
      }, this).join('<br>')
    },

    deleteCNAME () {
      ZoneApi.deleteZoneService(this.origin, this.zoneId, this.zoneServices[0])
        .then(
          (response) => {
            this.$emit('update-my-services', response.data)
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
    }
  }
}
</script>
