import Vue from 'vue'
import VueRouter from 'vue-router'
import Home from '../views/Home.vue'

Vue.use(VueRouter)

const routes = [
  {
    path: '/',
    name: 'home',
    component: Home
  },
  {
    path: '/zones',
    name: 'zones',
    component: function () {
      return import(/* webpackChunkName: "about" */ '../views/zone-list.vue')
    }
  },
  {
    path: '/zones/:zone',
    name: 'zone',
    component: function () {
      return import(/* webpackChunkName: "about" */ '../views/zone.vue')
    }
  }
]

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes
})

export default router
