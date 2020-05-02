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
    path: '/domains',
    name: 'domains',
    component: function () {
      return import(/* webpackChunkName: "domain-list" */ '../views/domain-list.vue')
    }
  },
  {
    path: '/domains/:domain',
    component: function () {
      return import(/* webpackChunkName: "domain" */ '../views/domain.vue')
    },
    children: [
      {
        path: '',
        name: 'domain-source',
        component: function () {
          return import(/* webpackChunkName: "domain-source" */ '../views/domain-source.vue')
        }
      },
      {
        path: 'services',
        name: 'zone-services',
        component: function () {
          return import(/* webpackChunkName: "zone-services" */ '../views/zone-services.vue')
        }
      }
    ]
  },
  {
    path: '/domains/:domain/new',
    name: 'domain-new',
    component: function () {
      return import(/* webpackChunkName: "domain-new" */ '../views/domain-new.vue')
    }
  },
  {
    path: '/sources',
    name: 'source-list',
    component: function () {
      return import(/* webpackChunkName: "source-list" */ '../views/source-list.vue')
    }
  },
  {
    path: '/sources/:source',
    component: function () {
      return import(/* webpackChunkName: "source" */ '../views/source.vue')
    },
    children: [
      {
        path: '',
        name: 'source-update',
        component: function () {
          return import(/* webpackChunkName: "source-update" */ '../views/source-update.vue')
        }
      }
    ]
  },
  {
    path: '/tools/client',
    name: 'tools-client',
    component: function () {
      return import(/* webpackChunkName: "tools-client" */ '../views/tools-client.vue')
    }
  },
  {
    path: '/tools/client/:domain',
    name: 'tools-client-domain',
    component: function () {
      return import(/* webpackChunkName: "tools-client" */ '../views/tools-client.vue')
    }
  },
  {
    path: '/zones/:domain/records',
    name: 'zone-records',
    component: function () {
      return import(/* webpackChunkName: "zone-records" */ '../views/zone-records.vue')
    }
  }
]

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes
})

export default router
