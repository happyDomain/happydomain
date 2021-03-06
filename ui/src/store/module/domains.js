// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

import Vue from 'vue'
import DomainsApi from '@/api/domains'
import { domainCompare } from '@/utils/domainCompare'

export default {
  namespaced: true,

  state: {
    all: null,
    detailed: {}
  },

  getters: {
    domains_getAll: state => state.all,
    domains_getDetailed: state => state.detailed,
    sortedDomains: state => state.all
  },

  actions: {
    dropDomain ({ commit }, domain) {
      commit('removeDomain', domain)
    },

    getAllMyDomains ({ commit }) {
      DomainsApi.listDomains()
        .then(
          response => {
            commit('setDomains', response.data)
          })
      // TODO: handle errors here
    },

    getDomainDetails ({ commit }, domain) {
      DomainsApi.getDomain(domain)
        .then(
          response => {
            const details = response.data
            commit('setDomainDetailed', { domain, details })
          }
        )
      // TODO: handle errors here
    },

    updateDomain ({ commit }, domain) {
      DomainsApi.updateDomain(domain)
        .then(
          response => {
            const details = response.data
            commit('setDomainDetailed', { domain: domain.domain, details })
          }
        )
    }
  },

  mutations: {
    removeDomain (state, domain) {
      Vue.delete(state.detailed, domain)
      Vue.set(state, 'all', null)
    },

    setDomainDetailed (state, { domain, details }) {
      Vue.set(state.detailed, domain, details)
    },

    setDomains (state, domains) {
      domains.sort(function (a, b) { return domainCompare(a.domain, b.domain) })
      Vue.set(state, 'all', domains)
    }
  }
}
