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
  <b-modal
    id="modal-manageDnGroups"
    scrollable
    size="lg"
    :title="$t('domaingroups.manage')"
    @cancel="handleModalCancel"
    @ok="handleModalOk"
  >
    <b-form
      class="float-right"
      inline
      @submit="addGroup"
    >
      <b-form-input
        id="newgroup"
        v-model="newgroup"
        :placeholder="$t('domaingroups.new')"
        required
      />
      <b-button
        type="submit"
        variant="primary"
      >
        <i18n path="common.add" />
      </b-button>
    </b-form>
    <div class="clearfix mb-2" />
    <h-zone-list
      :domains="sortedDomains"
    >
      <template #badges="{ domain }">
        <div class="float-right">
          <b-form-select
            v-model="domain.group"
            :options="groups"
            @change="groupChanged($event, domain)"
          />
        </div>
      </template>
    </h-zone-list>
  </b-modal>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
  name: 'HDomaingroupModal',

  components: {
    hZoneList: () => import('@/components/ZoneList')
  },

  data () {
    return {
      diff: { },
      groups: [],
      newgroup: ''
    }
  },

  computed: {
    ...mapGetters('domains', ['sortedDomains'])
  },

  mounted () {
    const groups = { }

    this.sortedDomains.forEach((domain, index) => {
      if (groups[domain.group] === undefined) {
        groups[domain.group] = []
      }

      groups[domain.group].push(domain)
    })

    const g = []
    Object.keys(groups).forEach((d) => {
      g.push({ text: d === 'undefined' ? this.$t('domaingroups.no-group') : d, value: d })
    })
    this.groups = g
  },

  methods: {
    addGroup (e) {
      e.preventDefault()
      this.groups.push(this.newgroup)
      this.newgroup = ''
    },

    groupChanged (gr, domain) {
      this.diff[domain.id] = () => {
        this.$store.dispatch('domains/updateDomain', domain)
      }
    },

    handleModalCancel () {
      this.$store.dispatch('domains/getAllMyDomains')
    },

    handleModalOk () {
      Object.keys(this.diff).forEach((k) => this.diff[k]())
      this.$store.dispatch('domains/getAllMyDomains')
    },

    show () {
      this.$bvModal.show('modal-manageDnGroups')
    }
  }
}
</script>
