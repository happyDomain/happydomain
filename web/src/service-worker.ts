import { build, files, version } from '$service-worker';

// Create a unique cache name for this deployment
const CACHE = `cache-${version}`;

const ASSETS = [
    ...build, // the app itself
    ...files  // everything in `static`
];

self.addEventListener('install', (event) => {
    // Create a new cache and add all files to it
    async function addFilesToCache() {
        const cache = await caches.open(CACHE);
        await cache.addAll(ASSETS);
    }

    event.waitUntil(addFilesToCache());
});

self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        console.log("SKIP_WAITING");
        self.skipWaiting();
    }
});

self.addEventListener('activate', (event) => {
    console.log(`%c SW ${version} `, 'background: #ddd; color: #0000ff')

    // Remove previous cached data from disk
    async function deleteOldCaches() {
        for (const key of await caches.keys()) {
            if (key !== CACHE) {
                await caches.delete(key);
                console.log(`%c Cleared ${key}`, 'background: #333; color: #ff0000')
            }
        }
    }

    if (caches) {
        event.waitUntil(deleteOldCaches());
    }
});

self.addEventListener('fetch', (event) => {
    // ignore POST requests etc
    if (event.request.method !== 'GET') return;

    async function respond() {
        const url = new URL(event.request.url);
        const cache = await caches.open(CACHE);

        // `build`/`files` can always be served from the cache
        if (ASSETS.includes(url.pathname)) {
            if (url.search.length) {
                url.search = "";
                return cache.match(new Request(url.toString(), event.request));
            }
            return cache.match(event.request);
        }

        if (
            url.pathname.startsWith("/api/providers/_specs") ||
                url.pathname.startsWith("/api/service_specs/")
        ) {
            // cache first
            const responseFromCache = await caches.match(event.request);
            if (responseFromCache) {
                return responseFromCache;
            }

            const response = await fetch(event.request);
            if (response.status === 200) {
                cache.put(event.request, response.clone());
            }

            return response;
        }

        // for everything else, try the network first, but
        // fall back to the cache if we're offline
        try {
            const response = await fetch(event.request);

            if (response.status === 200 && url.pathname != "/api/auth") {
                cache.put(event.request, response.clone());
            }

            return response;
        } catch {
            return cache.match(event.request);
        }
    }

    event.respondWith(respond());
});
