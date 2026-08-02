import { fileURLToPath } from 'node:url';

import base from '../eslint.config.base.js';
import svelteConfig from './svelte.config.js';

export default base({
	svelteConfig,
	gitignorePath: fileURLToPath(new URL('./.gitignore', import.meta.url))
});
