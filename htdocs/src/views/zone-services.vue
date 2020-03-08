<template>
  <div>
    <div class="d-flex flex-row justify-content-around flex-wrap align-self-center">
      <div class="p-3 service" v-for="(svc, index) in services" v-bind:key="index">
        <img :src="svc.logo" :alt="svc.name">
        Utiliser {{ svc.name }}
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  data: function () {
    return {
      error: '',
      services: [],
      zone: {}
    }
  },

  mounted () {
    axios
      .get('/api/services')
      .then(response => (this.services = response.data))

    var myzone = this.$route.params.zone
    axios
      .get('/api/zones/' + myzone)
      .then(response => (this.zone = response.data))
  }
}
</script>

<style>
.services {
    display: flex;
    align-items: center;
    justify-content: center;
}
.service {
    box-shadow: 2px 2px black;
    border: 1px solid black;
    display: flex;
    flex-direction: column;
    align-items: center;
    margin: 2.5%;
    width: 20%;
    height: 150px;
    text-align: center;
    vertical-align: middle;
}
.service img {
    max-width: 100%;
    max-height: 90%;
    padding: 2%;
}
</style>
