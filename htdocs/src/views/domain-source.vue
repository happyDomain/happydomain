<template>
  <div>
    <h3 class="text-primary">Paramètres <router-link to="#" class="badge badge-info">Change</router-link></h3>
    <p>
      <span class="text-secondary">Server</span><br>
      <strong>{{ zone.server }}</strong>
    </p>
    <p>
      <span class="text-secondary">DDNS Key Name</span><br>
      <strong>{{ zone.keyname }}</strong>
    </p>
    <p>
      <span class="text-secondary">DDNS Key Algorithm</span><br>
      <strong>{{ zone.algorithm }}</strong>
    </p>
    <p>
      <span class="text-secondary">Storage facility</span><br>
      <strong>{{ zone.storage_facility }}</strong>
    </p>
  </div>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      zone: {}
    }
  },

  mounted () {
    var myzone = this.$route.params.zone
    axios
      .get('/api/zones/' + myzone)
      .then(response => (this.zone = response.data))
  }
}
</script>
