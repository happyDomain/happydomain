<!--
    Copyright or © or Copr. happyDNS (2021)

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
  <h-list
    button
    :items="groups"
    :is-active="(group) => group === selectedGroup"
    @click="selectGroup"
  >
    <template #default="{ item }">
      <i18n v-if="item === 'undefined'" path="domaingroups.no-group" tag="span" />
      <span v-else>
        {{ item }}
      </span>
    </template>
  </h-list>
</template>

<script>
export default {
  name: 'hDomaingroupList',

  components: {
    hList: () => import('@/components/hList')
  },

  props: {
    domains: {
      type: Array,
      default: null
    },
    selectedGroup: {
      type: String,
      default: ''
    }
  },

  computed: {
    groups () {
      const groups = {}

      this.domains.forEach((domain, index) => {
        if (groups[domain.group] === undefined) {
          groups[domain.group] = null
        }
      })

      return Object.keys(groups)
    }
  },

  methods: {
    selectGroup (group) {
      if (this.selectedGroup && this.selectedGroup === group) {
        this.$emit('group-selected', '')
      } else {
        this.$emit('group-selected', group)
      }
    }
  }
}
</script>
