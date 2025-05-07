import { sveltekit } from '@sveltejs/kit/vite';
import type { UserConfig } from 'vite';

const config: UserConfig = {
        server: {
            hmr: {
                port: 10000
            }
        },

	plugins: [sveltekit()]
};

export default config;
