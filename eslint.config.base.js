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
				// A leading underscore is how the code already says a binding is
				// there for its position, not for its value.
				'@typescript-eslint/no-unused-vars': [
					'error',
					{
						argsIgnorePattern: '^_',
						varsIgnorePattern: '^_',
						caughtErrorsIgnorePattern: '^_',
						destructuredArrayIgnorePattern: '^_'
					}
				],
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
			// Every service editor is handed the same props by the loader that
			// mounts it, which knows the contract while the service does not:
			// leaving out the ones a given editor has no use for would make each
			// signature differ from the next for no reason.
			files: ['**/lib/services/*/editor.svelte'],
			rules: {
				'svelte/no-unused-props': 'off',
				'@typescript-eslint/no-unused-vars': [
					'error',
					{
						varsIgnorePattern: '^(_|dn$|origin$|readonly$|type$)',
						argsIgnorePattern: '^_',
						caughtErrorsIgnorePattern: '^_',
						destructuredArrayIgnorePattern: '^_'
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
