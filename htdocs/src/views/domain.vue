<template>
  <b-container fluid>
    <b-row>
      <b-col cols="3" class="text-right" style="background-color: #EAFFEC">
        <router-link to="/domains/" class="btn font-weight-bolder"><b-icon icon="chevron-up"></b-icon></router-link>
      </b-col>
      <b-col cols="9">
        <h2 class="mt-3 mb-3">
          {{ domain.domain }}
        </h2>
      </b-col>
    </b-row>
    <b-alert variant="danger" :show="error.length != 0"><strong>Error:</strong> {{ error }}</b-alert>
    <div class="text-center" v-if="!domain && error.length == 0">
      <b-spinner label="Spinning"></b-spinner>
      <p>Loading the domain&nbsp;&hellip;</p>
    </div>
    <b-row>
      <b-col cols="3" style="background-color: #EAFFEC">
        <b-navbar class="flex-column">
          <b-nav pills vertical>
            <b-nav-item :to="'/domains/' + domain.domain" :active="$route.name == 'domain-source'">Domain source</b-nav-item>
            <b-nav-item :to="'/domains/' + domain.domain + '/services'" :active="$route.name == 'zone-services'">View services</b-nav-item>
            <b-nav-item :to="'/zones/' + domain.domain + '/records'" :active="$route.name == 'zone-records'">View records</b-nav-item>
            <b-nav-item :to="'/domain/' + domain.domain + '/monitoring'" :active="$route.name == 'domain-monitoring'">Monitoring</b-nav-item>
            <hr>
            <b-nav-form>
              <b-button type="button" @click="detachDomain()" variant="outline-danger"><b-icon icon="trash-fill"></b-icon> Stop managing this domain</b-button>
            </b-nav-form>
          </b-nav>
        </b-navbar>
      </b-col>
      <b-col cols="9" class="mb-5">
        <router-view :domain="domain"></router-view>
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
      domain: {}
    }
  },

  mounted () {
    var mydomain = this.$route.params.domain
    axios
      .get('/api/domains/' + mydomain)
      .then(response => (this.domain = response.data))
  },

  methods: {
    detachDomain () {
      axios
        .delete('/api/domains/' + this.domain.domain)
        .then(response => (
          this.$router.push('/domains/')
        ))
    }
  }
}
</script>
