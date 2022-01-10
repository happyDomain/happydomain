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
  <div>
    <div v-if="isLoading" class="d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" :label="$t('common.spinning')" class="my-2 mr-3" /> <i18n :path="loadingStr" />
    </div>
    <slot v-else-if="domains.length === 0" name="no-domain" />
    <div v-else v-for="(domains, group) in groups" :key="group" :class="Object.keys(groups).length != 1?'border-top':''" style="margin-top: 1.4em">
      <div v-if="Object.keys(groups).length != 1" class="text-center" style="height: 1em">
        <h3 class="d-inline-block px-1" style="background: white; position: relative; top: -.65em">
          <i18n v-if="group === 'undefined'" path="domaingroups.no-group" />
          <span v-else>{{ group }}</span>
        </h3>
      </div>
      <h-list
        :items="domains"
        :button="button"
        @click="$emit('click', $event)"
      >
        <template #default="{ item }">
          <div class="text-monospace">
            <div class="d-inline-block text-center" style="width: 50px;">
              <img v-if="providers_getAll[item.id_provider]" :src="'/api/providers/_specs/' + providers_getAll[item.id_provider]._srctype + '/icon.png'" :alt="providers_getAll[item.id_provider]._srctype" :title="providers_getAll[item.id_provider]._srctype" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
            </div>
            {{ item.domain }}
          </div>
        </template>
        <template #badges="{ item }">
          <slot name="badges" :domain="item" />
        </template>
      </h-list>
    </div>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
  name: 'ZoneList',

  components: {
    hList: () => import('@/components/hList')
  },

  props: {
    button: {
      type: Boolean,
      default: false
    },
    displayByGroups: {
      type: Boolean,
      default: false
    },
    domains: {
      type: Array,
      default: null
    },
    loadingStr: {
      type: String,
      default: 'wait.retrieving-domains'
    },
    parentIsLoading: {
      type: Boolean,
      default: false
    }
  },

  computed: {
    groups () {
      if (!this.displayByGroups) {
        return { null: this.domains }
      }

      const groups = { }

      this.domains.forEach((domain, index) => {
        if (groups[domain.group] === undefined) {
          groups[domain.group] = []
        }

        groups[domain.group].push(domain)
      })

      return groups
    },

    isLoading () {
      return this.parentIsLoading || this.domains == null || this.providers_getAll == null
    },

    ...mapGetters('providers', ['providers_getAll'])
  }
}
</script>
