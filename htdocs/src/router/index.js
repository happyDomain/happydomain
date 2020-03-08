import Vue from 'vue'
import VueRouter from 'vue-router'

Vue.use(VueRouter)

const routes = [
  {
    path: '/',
    name: 'home',
    component: function () {
      return import(/* webpackChunkName: "Index" */ '@/views/Index.vue')
    }
  },
  {
    path: '/login',
    name: 'login',
    component: function () {
      return import(/* webpackChunkName: "login" */ '../views/login.vue')
    }
  },
  {
    path: '/join',
    name: 'signup',
    component: function () {
      return import(/* webpackChunkName: "signup" */ '../views/signup.vue')
    }
  },
  {
    path: '/zones',
    name: 'zones',
    component: function () {
      return import(/* webpackChunkName: "zone-list" */ '../views/zone-list.vue')
    }
  },
  {
    path: '/zones/:zone',
    component: function () {
      return import(/* webpackChunkName: "zone" */ '../views/zone.vue')
    },
    children: [
      {
        path: '',
        name: 'zone',
        component: function () {
          return import(/* webpackChunkName: "zone" */ '../views/zone-details.vue')
        }
      },
      {
        path: 'services',
        name: 'zone-services',
        component: function () {
          return import(/* webpackChunkName: "zone" */ '../views/zone-services.vue')
        }
      }
    ]
  },
  {
    path: '/zones/:zone/records',
    name: 'zone-records',
    component: function () {
      return import(/* webpackChunkName: "zone" */ '../views/zone-records.vue')
    }
  }
]

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes
})

export default router