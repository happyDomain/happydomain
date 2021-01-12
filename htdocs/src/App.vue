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
  <div id="app">
    <b-navbar :class="user_isLogged?'p-0':''" style="position: relative; z-index: 15">
      <b-container>
        <b-navbar-brand class="navbar-brand" to="/">
          <h-logo height="25" />
        </b-navbar-brand>

        <b-navbar-nav v-if="user_isLogged" class="ml-auto">
          <b-nav-form>
            <h-help size="sm" />
          </b-nav-form>

          <b-nav-item-dropdown v-if="user_isLogged" right>
            <template slot="button-content">
              <b-button v-if="user_getSession.email !== '_no_auth'" size="sm" variant="dark">
                <b-icon icon="person" aria-hidden="true" /> {{ user_getSession.email }}
              </b-button>
              <b-button v-else size="sm" variant="secondary">
                {{ $t('menu.quick-menu') }}
              </b-button>
            </template>
            <b-dropdown-item to="/domains/">
              {{ $t('menu.my-domains') }}
            </b-dropdown-item>
            <b-dropdown-item to="/sources/">
              {{ $t('menu.my-sources') }}
            </b-dropdown-item>
            <b-dropdown-divider />
            <b-dropdown-item to="/resolver">
              {{ $t('menu.dns-resolver') }}
            </b-dropdown-item>
            <b-dropdown-divider />
            <b-dropdown-item to="/me">
              {{ $t('menu.my-account') }}
            </b-dropdown-item>
            <b-dropdown-divider v-if="user_getSession.email !== '_no_auth'" />
            <b-dropdown-item v-if="user_getSession.email !== '_no_auth'" @click="logout()">
              {{ $t('menu.logout') }}
            </b-dropdown-item>
          </b-nav-item-dropdown>
        </b-navbar-nav>
        <b-navbar-nav v-else class="ml-auto">
          <h-help class="mr-2" size="sm" variant="link" />

          <b-button class="mr-2" variant="info" to="/resolver">
            <b-icon icon="list" aria-hidden="true" /> {{ $t('menu.dns-resolver') }}
          </b-button>

          <b-button variant="outline-dark" to="/join">
            <b-icon icon="person-plus-fill" aria-hidden="true" /> {{ $t('menu.signup') }}
          </b-button>
          <b-button variant="primary" class="ml-2" to="/login">
            <b-icon icon="person-check" aria-hidden="true" /> {{ $t('menu.signin') }}
          </b-button>

          <b-nav-item-dropdown :text="$i18n.locale" right>
            <b-dropdown-item v-for="(lid, lang) in languages" :key="lid" @click="chLanguage(lang)">
              {{ languages[lang] }}
            </b-dropdown-item>
          </b-nav-item-dropdown>
        </b-navbar-nav>
      </b-container>
    </b-navbar>

    <router-view class="flex-grow-1" />

    <b-toaster name="b-toaster-content-right" style="position: fixed; top: 70px; right: 0; z-index: 1042; min-width: 30vw;" />

    <a id="voxpeople" :href="'https://framaforms.org/quel-est-votre-avis-sur-happydns-1610366701?u=' + (user_isLogged?user_getSession.id:0) + '&amp;p=' + $router.history.current.name + '&amp;l=' + $i18n.locale" target="_blank">
      <b-icon icon="chat-right-text" style="width: 2vw; height: 2vw;" />
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
import Languages from '@/mixins/languages'

export default {
  components: {
    hHelp: () => import('@/components/hHelp')
  },

  mixins: [Languages],

  data: function () {
    return {
      alreadyShownUpdate: false
    }
  },

  computed: {
    ...mapGetters('user', ['user_getSession', 'user_getSettings', 'user_isLogged'])
  },

  watch: {
    user_getSettings: function (settings) {
      if (settings && settings.language && this.$i18n.locale !== settings.language) {
        this.$i18n.locale = settings.language
      }
    }
  },

  created () {
    this.$store.dispatch('sourceSpecs/getAllSourceSpecs')
    this.$store.dispatch('user/retrieveSession').catch(() => {})
  },

  mounted () {
    setInterval(function (vm) { vm.checkForUpdate() }, 360000, this)
    setTimeout(function (vm) { vm.checkForUpdate() }, 42000, this)
  },

  methods: {
    logout () {
      this.$store.dispatch('user/logout')
        .then(
          (response) => {
            this.$router.push('/')
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.logout'),
                autoHideDelay: 5000,
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    },

    chLanguage (lang) {
      this.$i18n.locale = lang
    },

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
