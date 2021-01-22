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
  <b-container fluid>
    <div v-if="!domain" class="mt-5 text-center">
      <b-spinner label="Spinning" />
      <p>{{ $t('wait.loading') }}</p>
    </div>
    <b-row v-else style="min-height: inherit">
      <b-col sm="4" md="3" class="bg-light pt-2 pb-2 sticky-top d-flex flex-column justify-content-between" style="height: 100vh; overflow-y: auto; z-index: 5">
        <b-row>
          <b-col class="pr-0" sm="auto">
            <router-link to="/domains/" class="btn font-weight-bolder">
              <b-icon icon="chevron-up" />
            </router-link>
          </b-col>
          <b-col class="pl-0">
            <b-form-select :value="domain.domain" :options="sortedDomains" value-field="domain" text-field="domain" @input="changeDomain($event)" />
          </b-col>
        </b-row>

        <b-form class="mt-3">
          <label class="font-weight-bolder" for="zhistory">{{ $t('domains.history') }}:</label>
          <b-form-select id="zhistory" v-model="selectedHistory" :options="domain.zone_history" value-field="id" text-field="last_modified" />
        </b-form>

        <b-button-group class="mt-3 w-100">
          <b-button size="sm" variant="outline-info" :title="$t('domains.actions.view')" @click="viewZone">
            <b-icon icon="list-ul" aria-hidden="true" /><br>
            {{ $t('domains.actions.view') }}
          </b-button>
          <b-button v-if="domain.zone_history.length && selectedHistory === domain.zone_history[0].id" size="sm" variant="success" :title="$t('domains.actions.propagate')" @click="showDiff">
            <b-icon icon="cloud-upload" aria-hidden="true" /><br>
            {{ $t('domains.actions.propagate') }}
          </b-button>
          <b-button v-else size="sm" variant="warning" :title="$t('domains.actions.rollback')" @click="showDiff">
            <b-icon icon="cloud-upload" aria-hidden="true" /><br>
            {{ $t('domains.actions.rollback') }}
          </b-button>
        </b-button-group>

        <hr>

        <b-form class="mt-3">
          <label class="font-weight-bolder">{{ $t('domains.views.as') }}</label>
          <b-button-group class="w-100">
            <b-button :variant="displayFormat === 'grid' ? 'secondary' : 'outline-secondary'" :title="$t('domains.views.grid.title')" @click="toogleGridView()">
              <b-icon icon="grid-fill" aria-hidden="true" /><br>
              {{ $t('domains.views.grid.label') }}
            </b-button>
            <b-button :variant="displayFormat === 'list' ? 'secondary' : 'outline-secondary'" :title="$t('domains.views.list.title')" @click="toogleListView()">
              <b-icon icon="list-ul" aria-hidden="true" /><br>
              {{ $t('domains.views.list.label') }}
            </b-button>
            <b-button :variant="displayFormat === 'records' ? 'secondary' : 'outline-secondary'" :title="$t('domains.views.records.title')" @click="toogleRecordsView()">
              <b-icon icon="menu-button-wide-fill" aria-hidden="true" /><br>
              {{ $t('domains.views.records.label') }}
            </b-button>
          </b-button-group>
        </b-form>

        <hr>

        <b-button class="w-100" type="button" variant="outline-danger" @click="detachDomain()">
          <b-icon icon="trash-fill" /> {{ $t('domains.stop') }}
        </b-button>

        <b-form v-if="sources_getAll && sources_getAll[domain.id_source]" class="mt-5">
          <label class="font-weight-bolder">{{ $t('domains.view.source') }}:</label>
          <div class="pr-2 pl-2">
            <b-button class="p-3 w-100 text-left" type="button" variant="outline-info" @click="goToSource()">
              <div class="d-inline-block text-center" style="width: 50px;">
                <img :src="'/api/source_specs/' + sources_getAll[domain.id_source]._srctype + '/icon.png'" :alt="sources_getAll[domain.id_source]._srctype" :title="sources_getAll[domain.id_source]._srctype" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
              </div>
              {{ sources_getAll[domain.id_source]._comment }}
            </b-button>
          </div>
        </b-form>
      </b-col>

      <b-col sm="8" md="9" class="mb-5">
        <div v-if="importInProgress" class="mt-4 text-center">
          <b-spinner :label="$t('common.spinning')" />
          <p>{{ $t('wait.importing') }}</p>
        </div>
        <h-subdomain-list v-else-if="selectedHistory && domain.zone_history.length > 0" :display-format="displayFormat" :domain="domain" :zone-id="selectedHistory" />
      </b-col>
    </b-row>

    <b-modal id="modal-viewZone" :title="$t('domains.view.title')" size="lg" scrollable ok-only :ok-disabled="zoneContent === null">
      <div v-if="zoneContent === null" class="my-2 text-center">
        <b-spinner label="Spinning" />
        <p>{{ $t('wait.formating') }}</p>
      </div>
      <pre v-else style="overflow: initial">{{ zoneContent }}</pre>
    </b-modal>

    <b-modal id="modal-applyZone" size="lg" scrollable @ok.prevent="applyDiff()">
      <template #modal-title>
        <i18n path="domains.view.description" tag="span">
          <span class="text-monospace">{{ domain.domain }}</span>
        </i18n>
      </template>
      <template #modal-footer="{ ok, cancel }">
        <div v-if="zoneDiffAdd || zoneDiffDel">
          <span v-if="zoneDiffAdd" class="text-success">
            {{ $tc('domains.apply.additions', (zoneDiffAdd || []).length) }}
          </span>
          &ndash;
          <span class="text-danger">
            {{ $tc('domains.apply.deletions', (zoneDiffDel || []).length) }}
          </span>
        </div>
        <b-button variant="secondary" @click="cancel()">
          {{ $t('common.cancel') }}
        </b-button>
        <b-button variant="success" :disabled="propagationInProgress || (zoneDiffAdd === null && zoneDiffDel === null)" @click="ok()">
          <b-spinner v-if="propagationInProgress" label="Spinning" />
          {{ $t('domains.apply.button') }}
        </b-button>
      </template>
      <div v-if="zoneDiffAdd === null && zoneDiffDel === null" class="my-2 text-center">
        <b-spinner label="Spinning" />
        <p>{{ $t('wait.exporting') }}</p>
      </div>
      <div v-for="(line, n) in zoneDiffAdd" :key="'a' + n" class="text-monospace text-success" style="white-space: nowrap">
        +{{ line }}
      </div>
      <div v-for="(line, n) in zoneDiffDel" :key="'d' + n" class="text-monospace text-danger" style="white-space: nowrap">
        -{{ line }}
      </div>
    </b-modal>
  </b-container>
</template>

<script>
import { mapGetters } from 'vuex'
import DomainsApi from '@/api/domains'
import ZonesApi from '@/api/zones'

export default {
  components: {
    hSubdomainList: () => import('@/components/hSubdomainList')
  },

  data: function () {
    return {
      displayFormat: 'grid',
      importInProgress: false,
      propagationInProgress: false,
      selectedHistory: null,
      zoneContent: null,
      zoneDiffAdd: null,
      zoneDiffDel: null
    }
  },

  computed: {
    domain () {
      return this.domains_getDetailed[this.$route.params.domain]
    },

    ...mapGetters('domains', ['domains_getDetailed', 'sortedDomains']),
    ...mapGetters('sources', ['sources_getAll'])
  },

  watch: {
    domain: function () {
      this.pullDomain()
      if (!this.selectedHistory && this.domain.zone_history.length !== 0) {
        this.selectedHistory = this.domain.zone_history[0].id
      }
    }
  },

  created () {
    if (localStorage && localStorage.getItem('displayFormat')) {
      this.displayFormat = localStorage.getItem('displayFormat')
    }
    this.$store.dispatch('domains/getAllMyDomains')
    this.$store.dispatch('sources/getAllMySources')
    this.updateDomainInfo()
  },

  methods: {
    pullDomain () {
      if (this.domain.zone_history === null || this.domain.zone_history.length === 0) {
        this.importZone()
      } else {
        this.selectedHistory = this.domain.zone_history[0].id
      }
    },

    importZone () {
      this.importInProgress = true
      ZonesApi.importZone(this.domain.domain)
        .then(
          (response) => {
            this.importInProgress = false
            this.updateDomainInfo()
            this.selectedHistory = response.data.id
          }
        )
    },

    changeDomain (newDomain) {
      this.$router.push('/domains/' + encodeURIComponent(newDomain))
      this.updateDomainInfo()
    },

    goToSource () {
      this.$router.push('/sources/' + encodeURIComponent(this.domain.id_source))
    },

    detachDomain () {
      this.$bvModal.msgBoxConfirm(this.$t('domains.alert.remove', { domain: this.domain.domain }), {
        title: this.$t('domains.removal'),
        size: 'lg',
        okVariant: 'danger',
        okTitle: this.$t('domains.discard'),
        cancelVariant: 'outline-secondary',
        cancelTitle: this.$t('domains.view.cancel-title')
      })
        .then(value => {
          if (value) {
            DomainsApi.detachDomain(this.domain.domain)
              .then(
                () => {
                  this.$store.dispatch('domains/dropDomain', this.domain.domain)
                  this.$router.push('/domains/')
                }
              )
          }
        })
    },

    toogleGridView () {
      this.displayFormat = 'grid'
      if (localStorage) {
        localStorage.setItem('displayFormat', 'grid')
      }
    },

    toogleListView () {
      this.displayFormat = 'list'
      if (localStorage) {
        localStorage.setItem('displayFormat', 'list')
      }
    },

    toogleRecordsView () {
      this.displayFormat = 'records'
      if (localStorage) {
        localStorage.setItem('displayFormat', 'records')
      }
    },

    showDiff () {
      this.zoneDiffAdd = null
      this.zoneDiffDel = null
      this.$bvModal.show('modal-applyZone')
      ZonesApi.diffZone(this.domain.domain, '@', this.selectedHistory)
        .then(
          (response) => {
            if (response.data.toAdd == null && response.data.toDel == null) {
              this.$bvModal.hide('modal-applyZone')
              this.$bvModal.msgBoxOk(this.$t('domains.apply.nochange'))
            } else {
              this.zoneDiffAdd = response.data.toAdd
              this.zoneDiffDel = response.data.toDel
            }
          },
          (error) => {
            this.$bvModal.hide('modal-applyZone')
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.occurs', { when: 'exporting the zone' }),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    applyDiff () {
      this.propagationInProgress = true
      ZonesApi.applyZone(this.domain.domain, this.selectedHistory)
        .then(
          (response) => {
            this.$bvModal.hide('modal-applyZone')
            this.propagationInProgress = false
            this.$bvToast.toast(
              this.$t('domains.apply.done.description'), {
                title: this.$t('domains.apply.done.title'),
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              }
            )
          },
          (error) => {
            this.propagationInProgress = false
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.occurs', { when: 'applying the zone' }),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    viewZone () {
      this.zoneContent = null
      this.$bvModal.show('modal-viewZone')
      ZonesApi.viewZone(this.domain.domain, this.selectedHistory)
        .then(
          (response) => {
            this.zoneContent = response.data
          },
          (error) => {
            this.$bvModal.hide('modal-viewZone')
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.occurs', { when: 'exporting the zone' }),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    updateDomainInfo () {
      this.$store.dispatch('domains/getDomainDetails', this.$route.params.domain)
    }
  }
}
</script>
