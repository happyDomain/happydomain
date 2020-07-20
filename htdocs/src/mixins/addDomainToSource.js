// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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

import axios from 'axios'

export default {
  data () {
    return {
      validatingNewDomain: false
    }
  },

  methods: {
    selectExistingSource (source, domain, redirect) {
      this.validatingNewDomain = true

      axios
        .post('/api/domains', {
          id_source: source._id,
          domain: domain
        })
        .then(
          (response) => {
            this.$root.$bvToast.toast(
              'Great! ' + response.data.domain + ' has been added. You can manage it right now.', {
                title: 'New domain attached to happyDNS!',
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              }
            )
            if (redirect) {
              this.$router.push('/domains/' + encodeURIComponent(response.data.domain))
            } else if (this.refreshDomains) {
              this.refreshDomains()
            } else {
              this.$emit('domainAdded', response.data)
            }
          },
          (error) => {
            this.validatingNewDomain = false
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when adding the domain!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    }
  }
}
