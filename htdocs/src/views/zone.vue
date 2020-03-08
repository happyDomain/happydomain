<template>
  <b-container class="mt-2">
    <b-row>
      <b-col cols="3" class="text-right" style="background-color: #EAFFEC">
        <router-link to="/zones/" class="btn font-weight-bolder"><b-icon icon="chevron-up"></b-icon></router-link>
      </b-col>
      <b-col cols="9">
        <h2 class="mt-2 mb-3">
          {{ zone.domain }}
        </h2>
      </b-col>
    </b-row>
    <b-alert variant="danger" :show="error.length != 0"><strong>Error:</strong> {{ error }}</b-alert>
    <div class="text-center" v-if="!zone && error.length == 0">
      <b-spinner label="Spinning"></b-spinner>
      <p>Loading the zone&nbsp;&hellip;</p>
    </div>
    <b-row>
      <b-col cols="3" style="background-color: #EAFFEC">
        <b-navbar class="flex-column">
          <b-nav pills vertical>
            <b-nav-item :to="'/zones/' + zone.domain" :active="$route.name == 'zone'">My zone</b-nav-item>
            <b-nav-item :to="'/zones/' + zone.domain + '/services'" :active="$route.name == 'zone-services'">View services</b-nav-item>
            <b-nav-item :to="'/zones/' + zone.domain + '/records'" :active="$route.name == 'zone-records'">View records</b-nav-item>
            <b-nav-item :to="'/zones/' + zone.domain + '/monitoring'" :active="$route.name == 'zone-monitoring'">Monitoring</b-nav-item>
            <hr>
            <b-nav-form>
              <b-button type="button" @click="deleteZone(zone)" variant="outline-danger"><b-icon icon="trash-fill"></b-icon> Delete my zone</b-button>
            </b-nav-form>
          </b-nav>
        </b-navbar>
      </b-col>
      <b-col cols="9">
        <router-view></router-view>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      error: '',
      zone: {}
    }
  },

  mounted () {
    var myzone = this.$route.params.zone
    axios
      .get('/api/zones/' + myzone)
      .then(response => (this.zone = response.data))
  },

  methods: {
    deleteZone (zone) {
      axios
        .delete('/api/zones/' + zone.domain)
        .then(response => (
          axios
            .get('/api/zones')
            .then(response => {
              this.zones = response.data
              this.$router.go('/zones/')
            })
        ))
    }
  }
}
</script>
