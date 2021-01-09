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
  <component :is="displayFormat === 'grid' ? 'b-card' : 'b-list-group'" v-if="!service || services[service._svctype]" :class="displayFormat !== 'list' ? 'card-hover' : ''" :style="(!service ? 'border-style: dashed; ' : '') + (displayFormat === 'grid' ? 'width: 32%; min-width: 225px; margin-bottom: 1em; cursor: pointer;' : displayFormat === 'records' ? 'margin-bottom: .5em; cursor: pointer;' : '')" no-body>
    <b-card-body v-if="displayFormat === 'grid'" @click="$emit('show-service-window', service)">
      <div v-if="service" class="float-right">
        <b-badge v-for="(categorie, idcat) in services[service._svctype].categories" :key="idcat" variant="gray" class="ml-1">
          {{ categorie }}
        </b-badge>
      </div>
      <b-card-title v-if="service">
        {{ services[service._svctype].name }}
      </b-card-title>
      <b-card-title v-else>
        <b-icon icon="plus" /> New service
      </b-card-title>
      <b-card-sub-title v-if="service" class="mb-2">
        {{ services[service._svctype].description }}
      </b-card-sub-title>
      <b-card-sub-title v-else class="mb-2">
        Click here to add a new service to this subdomain.
      </b-card-sub-title>
      <b-card-text>
        <span v-if="service && service._comment">{{ service._comment }}</span>
      </b-card-text>
    </b-card-body>

    <b-list-group-item v-else-if="displayFormat === 'list'" button @click="toogleShowDetails()">
      <strong :title="services[service._svctype].description">{{ services[service._svctype].name }}</strong> <span v-if="service._comment" class="text-muted">{{ service._comment }}</span>
      <span v-if="services[service._svctype].comment" class="text-muted">{{ services[service._svctype].comment }}</span>
      <b-badge v-for="(categorie, idcat) in services[service._svctype].categories" :key="idcat" variant="gray" class="float-right ml-1">
        {{ categorie }}
      </b-badge>
    </b-list-group-item>
    <b-list-group-item v-if="showDetails">
      <h-editable-service edit-toolbar :origin="origin" :service="service" :services="services" :zone-id="zoneId" @update-my-services="$emit('update-my-services', $event)" />
    </b-list-group-item>

    <b-list-group-item v-else-if="displayFormat === 'records'" @click="$emit('show-service-window', service)">
      <strong :title="services[service._svctype].description">{{ services[service._svctype].name }}</strong> <span v-if="service._comment" class="text-muted">{{ service._comment }}</span>
      <span v-if="services[service._svctype].comment" class="text-muted">{{ services[service._svctype].comment }}</span>
      <b-badge v-for="(categorie, idcat) in services[service._svctype].categories" :key="idcat" variant="gray" class="float-right ml-1">
        {{ categorie }}
      </b-badge>
    </b-list-group-item>
    <b-list-group-item v-if="displayFormat === 'records' && serviceRecords" class="p-0">
      <table class="table table-hover table-bordered table-striped table-sm m-0">
        <tbody>
          <h-record v-for="(rr, irr) in serviceRecords" :key="irr" :record="rr" />
        </tbody>
      </table>
    </b-list-group-item>
  </component>
</template>

<script>
import ZoneApi from '@/api/zones'

export default {
  name: 'HDomainService',

  components: {
    hEditableService: () => import('@/components/hEditableService'),
    hRecord: () => import('@/components/hLinedRecord')
  },

  props: {
    displayFormat: {
      type: String,
      default: 'grid'
    },
    origin: {
      type: String,
      required: true
    },
    service: {
      type: Object,
      default: null
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
      serviceRecords: null,
      showDetails: false
    }
  },

  watch: {
    displayFormat: function (df) {
      if (df === 'records' && !this.serviceRecords) {
        this.updateServiceRecords()
      }
    },

    service: function (svc) {
      if (this.serviceRecords) {
        this.updateServiceRecords()
      }
    }
  },

  mounted () {
    this.updateServiceRecords()
  },

  methods: {
    toogleShowDetails () {
      this.showDetails = !this.showDetails
    },

    updateServiceRecords () {
      if (this.displayFormat === 'records') {
        ZoneApi.getServiceRecords(this.origin, this.zoneId, this.service)
          .then(response => {
            this.serviceRecords = response.data
          })
      }
    }
  }
}
</script>
