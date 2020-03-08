<template>
  <component :is="homeComponent"></component>
</template>

<script>
import Home from '@/views/Home'
import ZoneList from '@/views/zone-list'

export default {
  data () {
    return {
      homeComponent: sessionStorage.token !== undefined ? ZoneList : Home
    }
  }
}
</script>
