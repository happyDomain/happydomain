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

import { setCacheNameDetails, clientsClaim } from 'workbox-core'
import { createHandlerBoundToURL, precacheAndRoute } from 'workbox-precaching'
import { NavigationRoute, registerRoute } from 'workbox-routing'
import * as strategies from 'workbox-strategies'
import { ExpirationPlugin } from 'workbox-expiration'

setCacheNameDetails({ prefix: 'happyDNS' })

const LATEST_VERSION = 'v0.9'

self.addEventListener('activate', (event) => {
  console.log(`%c ${LATEST_VERSION} `, 'background: #ddd; color: #0000ff')
  if (caches) {
    caches.keys().then((arr) => {
      arr.forEach((key) => {
        if (key.indexOf('happyDNS-precache') < -1) {
          caches.delete(key).then(() => console.log(`%c Cleared ${key}`, 'background: #333; color: #ff0000'))
        } else {
          caches.open(key).then((cache) => {
            cache.match('version').then((res) => {
              if (!res) {
                cache.put('version', new Response(LATEST_VERSION, { status: 200, statusText: LATEST_VERSION }))
              } else if (res.statusText !== LATEST_VERSION) {
                caches.delete(key).then(() => console.log(`%c Cleared Cache ${LATEST_VERSION}`, 'background: #333; color: #ff0000'))
              } else console.log(`%c Great you have the latest version ${LATEST_VERSION}`, 'background: #333; color: #00ff00')
            })
          })
        }
      })
    })
  }
})

self.skipWaiting()
clientsClaim()

precacheAndRoute(self.__WB_MANIFEST)

const handler = createHandlerBoundToURL('/index.html')
const navigationRoute = new NavigationRoute(handler, { denylist: [/^\/api\//, /^\/fr\//, /^\/en\//, /^\/img\/screenshots\//] })
registerRoute(navigationRoute)

registerRoute(
  '/api/service_specs',
  new strategies.CacheFirst({
    cacheName: 'service-spec-cache',
    plugins: [
      new ExpirationPlugin({
        maxAgeSeconds: 24 * 60 * 60
      })
    ]
  })
)
registerRoute(
  /\/api\/service_specs\/.*/,
  new strategies.CacheFirst({
    cacheName: 'service-specs-cache',
    plugins: [
      new ExpirationPlugin({
        maxEntries: 100,
        maxAgeSeconds: 24 * 60 * 60
      })
    ]
  })
)

registerRoute(
  '/api/providers/_specs',
  new strategies.CacheFirst({
    cacheName: 'source-spec-cache',
    plugins: [
      new ExpirationPlugin({
        maxAgeSeconds: 24 * 60 * 60
      })
    ]
  })
)
registerRoute(
  /\/api\/providers\/_specs\/.*/,
  new strategies.CacheFirst({
    cacheName: 'source-specs-cache',
    plugins: [
      new ExpirationPlugin({
        maxEntries: 100,
        maxAgeSeconds: 24 * 60 * 60
      })
    ]
  })
)
