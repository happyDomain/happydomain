import { sveltekit } from '@sveltejs/kit/vite';
import { searchForWorkspaceRoot } from 'vite';
import type { UserConfig } from 'vite';

const config: UserConfig = {
        server: {
            fs: {
                allow: [
                    searchForWorkspaceRoot(process.cwd()),
                    searchForWorkspaceRoot(process.cwd() + "/../web"),
                ],
            },
            port: 5174,
            hmr: {
                port: 10001
            }
        },

	plugins: [sveltekit()]
};

export default config;
