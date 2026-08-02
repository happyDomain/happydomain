import prettier from 'eslint-config-prettier';
import js from '@eslint/js';
import { includeIgnoreFile } from '@eslint/compat';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import ts from 'typescript-eslint';

// The linted code lives in the workspaces, and each has a Svelte config and an
// ignore file of its own: they call this with theirs.
export default function config({ svelteConfig, gitignorePath }) {
	return ts.config(
		includeIgnoreFile(gitignorePath),
		js.configs.recommended,
		...ts.configs.recommended,
		...svelte.configs.recommended,
		prettier,
		...svelte.configs.prettier,
		{
			languageOptions: {
				globals: { ...globals.browser, ...globals.node }
			},
			rules: {
				'no-undef': 'off',
				'no-restricted-syntax': [
					'error',
					{
						selector: 'TSAsExpression[typeAnnotation.type="TSUnknownKeyword"]',
						message:
							'`as unknown as` turns off every check between the two types. Fix the type where it is declared, or narrow with a type guard.'
					}
				]
			}
		},
		{
			// Fixtures stand in for models a test does not fully build, and
			// generated clients answer to their generator, not to this rule.
			files: ['**/*.test.ts', '**/*.spec.ts', '**/*.gen.ts'],
			rules: { 'no-restricted-syntax': 'off' }
		},
		{
			files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
			languageOptions: {
				parserOptions: {
					projectService: true,
					extraFileExtensions: ['.svelte'],
					parser: ts.parser,
					svelteConfig
				}
			}
		}
	);
}
