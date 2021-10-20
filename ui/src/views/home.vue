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
  <b-container class="pt-4 pb-5">
    <i18n path="common.welcome" tag="h1" class="text-center mb-4">
      <h-logo height="40" />
    </i18n>

    <b-row>
      <b-col md="8" class="order-1 order-md-0">
        <h-zone-list
          ref="zlist"
          button
          :domains="filteredDomains"
          @click="showDomain($event)"
        >
          <template #badges>
            <b-badge variant="success">
              OK
            </b-badge>
          </template>
        </h-zone-list>
        <b-card
          v-if="filteredProvider && $refs.zlist && !$refs.zlist.isLoading"
          no-body
          :class="filteredDomains.length > 0 ? 'mt-4' : ''"
        >
          <template v-if="!noDomainsList" slot="header">
            <div class="d-flex justify-content-between">
              <i18n path="provider.provider">
                <em>{{ filteredProvider._comment }}</em>
              </i18n>
              <b-button
                v-if="$refs.newDomains && $refs.newDomains.listImportableDomains.length > 0"
                type="button"
                variant="secondary"
                size="sm"
                @click="$refs.newDomains.importAllDomains()"
              >
                {{ $t('provider.import-domains') }}
              </b-button>
            </div>
          </template>
          <h-provider-list-domains
            ref="newDomains"
            :provider="filteredProvider"
            show-domains-with-actions
            @no-domains-list-change="noDomainsList = $event"
          />
        </b-card>
        <h-list-group-input-new-domain
          v-if="$refs.zlist && !$refs.zlist.isLoading && (!filteredProvider || noDomainsList)"
          autofocus
          class="mt-2"
          :my-provider="filteredProvider"
        />
      </b-col>

      <b-col md="4" class="order-0 order-md-1">
        <b-card
          no-body
          class="mb-3"
        >
          <template slot="header">
            <div class="d-flex justify-content-between">
              <i18n path="provider.title" />
              <b-button
                size="sm"
                variant="light"
                @click="newProvider"
              >
                <b-icon icon="plus" />
              </b-button>
            </div>
          </template>
          <h-provider-list
            ref="providerList"
            no-label
            flush
            :selected-provider="filteredProvider"
            @provider-selected="filteredProvider = $event"
          />
        </b-card>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import { mapGetters } from 'vuex'

export default {

  components: {
    hListGroupInputNewDomain: () => import('@/components/hListGroupInputNewDomain'),
    hProviderListDomains: () => import('@/components/hProviderListDomains'),
    hProviderList: () => import('@/components/providerList'),
    hZoneList: () => import('@/components/ZoneList')
  },

  data: function () {
    return {
      noDomainsList: true,
      filteredGroup: null,
      filteredProvider: null
    }
  },

  computed: {
    filteredDomains () {
      if (this.sortedDomains && this.filteredProvider) {
        return this.sortedDomains.filter(d => d.id_provider === this.filteredProvider._id)
      } else {
        return this.sortedDomains
      }
    },

    ...mapGetters('domains', ['sortedDomains'])
  },

  watch: {
    sortedDomains: function (domains) {
      if (this.$route.params.provider === undefined && domains.length === 0) {
        this.$router.replace('/onboarding')
      }
    }
  },

  created () {
    this.$store.dispatch('domains/getAllMyDomains')
    this.$store.dispatch('providers/getAllMyProviders')
  },

  mounted () {
    if (this.$route.params.provider) {
      this.filteredProvider = { _id: parseInt(this.$route.params.provider) }
    }
  },

  methods: {
    newProvider () {
      this.$router.push('/providers/new')
    },

    showDomain (domain) {
      this.$router.push('/domains/' + encodeURIComponent(domain.domain))
    }
  }

}
</script>
