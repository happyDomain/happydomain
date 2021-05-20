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

export default {
  data () {
    return {
      form: null,
      nextIsWorking: false,
      previousIsWorking: false,
      settings: null,
      state: 0
    }
  },

  methods: {
    loadState (toState, recallid, cbSuccess, cbFail) {
      this.getFormSettings(toState, this.settings, recallid)
        .then(
          response => {
            this.previousIsWorking = false
            this.nextIsWorking = false
            if (response.data.form) {
              this.form = response.data.form
              this.state = toState
              if (response.data.redirect && window.location.pathname !== response.data.redirect) {
                this.$router.push(response.data.redirect)
              } else if (cbSuccess) {
                cbSuccess(toState)
              }
            } else if (response.data.Provider) {
              this.$root.$bvToast.toast(
                'Done', {
                  title: (response.data.Provider._comment ? response.data.Provider._comment : 'Your new provider') + ' has been ' + (this.settings._id ? 'updated' : 'added') + '.',
                  autoHideDelay: 5000,
                  variant: 'success',
                  toaster: 'b-toaster-content-right'
                }
              )
              if (response.data.redirect && window.location.pathname !== response.data.redirect) {
                this.$router.push(response.data.redirect)
              } else if (cbSuccess) {
                cbSuccess(toState, response.data.Provider)
              } else {
                this.$router.push('/providers/' + encodeURIComponent(response.data.Provider._id) + '/domains')
              }
            }
          },
          error => {
            this.previousIsWorking = false
            this.nextIsWorking = false
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'Something went wrong during provider configuration validation',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            if (cbFail) {
              cbFail(error.response.data)
            }
          })
    },

    nextState (bvModalEvt) {
      if (bvModalEvt) {
        bvModalEvt.preventDefault()
      }
      this.nextIsWorking = true
      if (this.form) {
        if (this.form.nextButtonLink !== undefined) {
          window.location = this.form.nextButtonLink
        } else if (this.form.nextButtonState === -1) {
          this.state = this.form.nextButtonState
          this.form = null
          this.nextButtonState = false
        } else if (this.form.nextButtonState) {
          this.loadState(
            this.form.nextButtonState,
            null,
            this.reactOnSuccess
          )
        } else {
          this.loadState(0)
        }
      } else {
        this.loadState(0)
      }
    },

    previousState () {
      this.previousIsWorking = true
      if (this.form.previousButtonState <= 0) {
        this.state = this.form.previousButtonState
        this.form = null
        this.previousIsWorking = false
      } else if (this.form.previousButtonState) {
        this.loadState(
          this.form.previousButtonState,
          null,
          this.reactOnSuccess
        )
      } else {
        this.loadState(0)
      }
    },

    resetSettings () {
      this.settings = {
        Provider: {},
        Service: {},
        _comment: '',
        redirect: null
      }
    },

    updateSettingsForm () {
      if (this.state >= 0) {
        this.loadState(this.state, this.$route.query.recall)
      }
    }
  }
}
