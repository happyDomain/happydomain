// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

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
    path: '/fr/',
    name: 'home-fr',
    component: function () {
      return import(/* webpackChunkName: "Index" */ '@/views/Index.vue')
    }
  },
  {
    path: '/en/',
    name: 'home-en',
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
    path: '/email-validation',
    name: 'email-validation',
    component: function () {
      return import(/* webpackChunkName: "signup" */ '../views/email-validation.vue')
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
          return import(/* webpackChunkName: "domain" */ '../views/domain-source.vue')
        }
      },
      {
        path: 'services',
        name: 'domain-services',
        component: function () {
          return import(/* webpackChunkName: "domain" */ '../views/domain-services.vue')
        }
      }
    ]
  },
  {
    path: '/domains/:domain/new',
    name: 'domain-new',
    component: function () {
      return import(/* webpackChunkName: "domain" */ '../views/domain-new.vue')
    }
  },
  {
    path: '/sources',
    name: 'source-list',
    component: function () {
      return import(/* webpackChunkName: "source" */ '../views/source-list.vue')
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
          return import(/* webpackChunkName: "source" */ '../views/source-update.vue')
        }
      },
      {
        path: 'domains',
        name: 'source-list-domains',
        component: function () {
          return import(/* webpackChunkName: "source" */ '../views/source-list-domains.vue')
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
      return import(/* webpackChunkName: "domain" */ '../views/zone-records.vue')
    }
  }
]

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes
})

export default router
