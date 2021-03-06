// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
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
import store from '@/store/index'

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
    },
    meta: {
      guest: true
    }
  },
  {
    path: '/join',
    name: 'signup',
    component: function () {
      return import(/* webpackChunkName: "signup" */ '../views/signup.vue')
    },
    meta: {
      guest: true
    }
  },
  {
    path: '/email-validation',
    name: 'email-validation',
    component: function () {
      return import(/* webpackChunkName: "signup" */ '../views/email-validation.vue')
    },
    meta: {
      guest: true
    }
  },
  {
    path: '/forgotten-password',
    name: 'forgotten-password',
    component: function () {
      return import(/* webpackChunkName: "forgotten-password" */ '../views/forgotten-password.vue')
    },
    meta: {
      guest: true
    }
  },
  {
    path: '/onboarding',
    name: 'onboarding',
    component: function () {
      return import(/* webpackChunkName: "home" */ '../views/onboarding.vue')
    },
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/me',
    name: 'me',
    component: function () {
      return import(/* webpackChunkName: "me" */ '../views/me.vue')
    },
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/domains',
    name: 'domains',
    component: function () {
      return import(/* webpackChunkName: "home" */ '../views/home.vue')
    },
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/domains/:domain',
    name: 'domain-abstract',
    component: function () {
      return import(/* webpackChunkName: "domain" */ '../views/domain.vue')
    },
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/domains/:domain/new',
    name: 'domain-new',
    component: function () {
      return import(/* webpackChunkName: "domain" */ '../views/domain-new.vue')
    },
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/providers',
    name: 'provider-list',
    component: function () {
      return import(/* webpackChunkName: "provider" */ '../views/provider-list.vue')
    },
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/providers/new',
    component: function () {
      return import(/* webpackChunkName: "provider-new" */ '../views/provider-new.vue')
    },
    children: [
      {
        path: '',
        name: 'provider-new-choice',
        component: function () {
          return import(/* webpackChunkName: "provider-new" */ '../views/provider-new-choice.vue')
        },
        meta: {
          requiresAuth: true
        }
      },
      {
        path: ':provider/:state',
        name: 'provider-new-state',
        component: function () {
          return import(/* webpackChunkName: "provider-new" */ '../views/provider-new-state.vue')
        },
        meta: {
          requiresAuth: true
        }
      }
    ]
  },
  {
    path: '/providers/:provider',
    name: 'provider-update',
    component: function () {
      return import(/* webpackChunkName: "provider" */ '../views/provider.vue')
    },
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/providers/:provider/domains',
    name: 'provider-update-domains',
    component: function () {
      return import(/* webpackChunkName: "home" */ '../views/home.vue')
    },
    meta: {
      requiresAuth: true
    }
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
    path: '/resolver',
    name: 'tools-resolver',
    component: function () {
      return import(/* webpackChunkName: "tools-client" */ '../views/tools-client.vue')
    }
  },
  {
    path: '/resolver/:domain',
    name: 'tools-resolver-domain',
    component: function () {
      return import(/* webpackChunkName: "tools-client" */ '../views/tools-client.vue')
    }
  },
  {
    path: '*',
    name: 'non-found',
    component: function () {
      return import(/* webpackChunkName: "not-found" */ '../views/404.vue')
    }
  }
]

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes
})

router.beforeEach((to, from, next) => {
  function routerProceed () {
    if (to.matched.some(record => record.meta.requiresAuth)) {
      if (store.getters['user/user_getSession'] == null) {
        next({
          path: '/login',
          params: { nextUrl: to.fullPath }
        })
      } else {
        next()
      }
    } else if (to.matched.some(record => record.meta.guest)) {
      if (store.getters['user/user_getSession'] == null) {
        next()
      } else {
        next({ name: 'home' })
      }
    } else {
      next()
    }
  }

  if (!store.state.user.initialized) {
    store.watch(
      (state) => state.user.initialized,
      (value) => {
        if (value) {
          routerProceed()
        }
      }
    )
  } else {
    routerProceed()
  }
})

export default router
