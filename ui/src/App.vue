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
  <div id="app">
    <h-header />

    <router-view class="flex-grow-1" />

    <b-toaster
      name="b-toaster-content-right"
      style="position: fixed; top: 70px; right: 0; z-index: 1042; min-width: 30vw;"
    />

    <a
      id="voxpeople"
      :href="'https://framaforms.org/quel-est-votre-avis-sur-happydns-1610366701?u=' + (user_isLogged?user_getSession.id:0) + '&amp;i=' + instancename + '&amp;p=' + $router.history.current.name + '&amp;l=' + $i18n.locale"
      target="_blank"
      :title="$t('common.survey')"
    >
      <b-icon
        icon="chat-right-text"
        style="width: 2vw; height: 2vw;"
      />
    </a>

    <footer class="pt-2 pb-2 bg-dark text-light">
      <b-container>
        <b-row>
          <b-col md="12" lg="6">
            &copy;
            <h-logo color="#fff" height="17" />
            2019-2021 All rights reserved
          </b-col>
          <b-col md="6" lg="3" />
          <b-col md="6" lg="3" />
        </b-row>
      </b-container>
    </footer>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
  components: {
    hHeader: () => import('@/components/hHeader')
  },

  data: function () {
    return {
      alreadyShownUpdate: false
    }
  },

  computed: {
    instancename () {
      return window.location.hostname
    },
    ...mapGetters('user', ['user_getSession', 'user_getSettings', 'user_isLogged'])
  },

  watch: {
    user_isLogged: function (isLogged) {
      if (isLogged && (this.$route.name === 'login' || this.$route.name === 'signup')) {
        this.$router.replace('/')
      }
    },
    user_getSettings: function (settings) {
      if (settings && settings.language && this.$i18n.locale !== settings.language) {
        this.$i18n.locale = settings.language
      }
    }
  },

  created () {
    this.$store.dispatch('providerSpecs/getAllProviderSpecs')
    this.$store.dispatch('user/retrieveSession').catch(() => {})
  },

  mounted () {
    setInterval(function (vm) { vm.checkForUpdate() }, 360000, this)
    setTimeout(function (vm) { vm.checkForUpdate() }, 42000, this)
  },

  methods: {
    checkForUpdate (timeout) {
      if (sessionStorage.getItem('happyUpdate') && !this.alreadyShownUpdate) {
        this.alreadyShownUpdate = true
        this.$bvToast.toast(
          this.$t('upgrade.content'), {
            title: this.$t('upgrade.title'),
            variant: 'primary',
            href: window.location.pathname,
            noAutoHide: true,
            toaster: 'b-toaster-content-right'
          }
        )
      }
    }
  }
}
</script>
