import { defineConfig } from 'eslint/config';
import grafanaConfig from '@grafana/eslint-config/flat.js';

export default defineConfig([
  {
    ignores: ['dist/**', 'node_modules/**', '.config/**'],
  },
  ...grafanaConfig,
  {
    rules: {
      'react/prop-types': 'off',
      'no-console': 'warn',
    },
  },
  {
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      parserOptions: {
        project: './tsconfig.json',
      },
    },
    rules: {
      '@typescript-eslint/no-deprecated': 'warn',
    },
  },
]);
