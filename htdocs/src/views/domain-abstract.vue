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
    <div v-if="importInProgress" class="mt-4 text-center">
      <b-spinner label="Spinning" />
      <p>Please wait while we are importing your domain&nbsp;&hellip;</p>
    </div>
    <div v-else-if="selectedHistory">
      <div class="mt-2 text-right">
        <b-button size="sm" class="mx-1" @click="importZone()"><b-icon icon="cloud-download" aria-hidden="true" /> Re-import</b-button>
        <b-button size="sm" class="mx-1" @click="viewZone()"><b-icon icon="list-ul" aria-hidden="true" /> View</b-button>
        <b-button size="sm" variant="success" class="mx-1" @click="showDiff()"><b-icon icon="cloud-upload" aria-hidden="true" /> Apply</b-button>
      </div>
      <h-subdomain-list :domain="domain" :zone-meta="selectedHistory" />
    </div>

    <b-modal id="modal-viewZone" title="View zone" size="lg" scrollable ok-only>
      <pre style="overflow: initial">{{ zoneContent }}</pre>
    </b-modal>

    <b-modal id="modal-applyZone" size="lg" scrollable @ok="applyDiff()">
      <template v-slot:modal-title>
        Review the modifications that will be applied to <span class="text-monospace">{{ domain.domain }}</span>
      </template>
      <div v-for="(line, n) in zoneDiffAdd" :key="'a' + n" class="text-monospace text-success" style="white-space: nowrap">
        +{{ line }}
      </div>
      <div v-for="(line, n) in zoneDiffDel" :key="'d' + n" class="text-monospace text-danger" style="white-space: nowrap">
        -{{ line }}
      </div>
    </b-modal>
  </div>
</template>

<script>
import axios from 'axios'
import ZoneApi from '@/services/ZoneApi'

export default {
  components: {
    hSubdomainList: () => import('@/components/hSubdomainList')
  },

  props: {
    domain: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      importInProgress: false,
      selectedHistory: null,
      zoneContent: null,
      zoneDiffAdd: null,
      zoneDiffDel: null
    }
  },

  watch: {
    domain: function () {
      this.pullDomain()
    }
  },

  created () {
    if (this.domain !== undefined && this.domain.domain !== undefined) {
      this.pullDomain()
    }
  },

  methods: {
    pullDomain () {
      if (this.domain.zone_history === null || this.domain.zone_history.length === 0) {
        this.importZone()
      } else {
        this.selectedHistory = this.domain.zone_history[0]
      }
    },

    importZone () {
      this.importInProgress = true
      axios
        .post('/api/domains/' + encodeURIComponent(this.domain.domain) + '/import_zone')
        .then(
          (response) => {
            this.importInProgress = false
            this.selectedHistory = response.data
            this.$parent.$emit('updateDomainInfo')
          }
        )
    },

    showDiff () {
      ZoneApi.diffZone(this.domain.domain, '@', this.selectedHistory.id)
        .then(
          (response) => {
            if (response.data.toAdd == null && response.data.toDel == null) {
              this.$bvModal.msgBoxOk('There is no changes to apply! Current zone is in sync with the server.')
            } else {
              this.zoneDiffAdd = response.data.toAdd
              this.zoneDiffDel = response.data.toDel
              this.$bvModal.show('modal-applyZone')
            }
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when applying the zone!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    applyDiff () {
      ZoneApi.applyZone(this.domain.domain, this.selectedHistory.id)
        .then(
          (response) => {
            this.$bvToast.toast(
              '!', {
                title: 'Zone applied successfully!',
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              }
            )
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when applying the zone!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    viewZone () {
      ZoneApi.viewZone(this.domain.domain, this.selectedHistory.id)
        .then(
          (response) => {
            this.zoneContent = response.data
            this.$bvModal.show('modal-viewZone')
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when applying the zone!',
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
