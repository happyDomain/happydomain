<template>
  <component :is="homeComponent"></component>
</template>

<script>
import ZoneList from '@/views/domain-list'

export default {
  beforeRouteEnter (to, from, next) {
    next(vm => {
      if (sessionStorage.token === undefined) {
        if (to.path === '/') {
          var preferedLang = navigator.language.split('-')[0] || process.env.VUE_APP_I18N_LOCALE || 'en'
          if (preferedLang !== 'en' && preferedLang !== 'fr') {
            preferedLang = 'en'
          }
          window.location.href = '/' + preferedLang + '/'
        } else {
          next({ path: '/login' })
        }
      } else {
        if (to.path !== '/') {
          next({ path: '/domains/' })
        }
      }
    })
  },

  data () {
    return {
      homeComponent: sessionStorage.token !== undefined ? ZoneList : ''
    }
  }
}
</script>
