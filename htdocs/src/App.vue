<template>
  <div id="app">
    <b-navbar size="lg" type="dark" variant="dark" sticky class="text-light">
      <b-navbar-brand class="navbar-brand" to="/">
        <img alt="LibreDNS" src="<%= BASE_URL %>img/logo.png" style="height: 30px">
      </b-navbar-brand>
      <button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#adminMenu" aria-controls="adminMenu" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
      </button>

      <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>

      <b-collapse id="nav-collapse" is-nav>
        <b-navbar-nav>
          <b-nav-item to="/zones">Zones</b-nav-item>
          <b-nav-item to="/users" disabled>Users</b-nav-item>
        </b-navbar-nav>
      </b-collapse>

      <b-navbar-nav class="ml-auto">
        <b-nav-item-dropdown right>
          <template slot="button-content"><div class="btn btn-sm btn-secondary">nemunaire</div></template>
          <b-dropdown-item>Some example text that's free-flowing within the dropdown menu.</b-dropdown-item>
          <b-dropdown-item href="#">Action</b-dropdown-item>
          <b-dropdown-item href="#">Another action</b-dropdown-item>
          <b-dropdown-item href="#">Something else here</b-dropdown-item>
        </b-nav-item-dropdown>
      </b-navbar-nav>
    </b-navbar>
    <div class="progress" style="background-color: #aee64e; height: 3px; border-radius: 0;">
      <div class="progress-bar bg-secondary" role="progressbar" style="width: 0%"></div>
    </div>

    <router-view/>
  </div>
</template>
