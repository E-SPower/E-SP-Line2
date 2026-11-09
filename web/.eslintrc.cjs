/* eslint-env node */
/**
 * ESLint config for the web front end — part of the merge gate.
 *
 * `npm run lint` is a CI job (.github/workflows/ci.yml) and a merge-gate item
 * (docs/developer-guide/pr-qa.md); .github/CONTRIBUTING.md §4 says it must be
 * clean. The base is the standard Vite + React + TypeScript template rule set:
 * type-aware-free TypeScript rules (fast enough for a hook and for CI), both
 * react-hooks rules, and react-refresh so a module gives the HMR boundary one
 * kind of export.
 *
 * Two notes for whoever edits this file:
 *
 * - `react-refresh/only-export-components` is an error, and the components that
 *   legitimately ship a sibling constant (`badgeVariants`, `controlClassName`,
 *   the `useTheme` hook) are listed by name in `allowExportNames` rather than
 *   scattered with per-file disable comments. Add a name here, not a disable.
 * - `@typescript-eslint/no-explicit-any` stays a warning against a fixed debt
 *   budget in package.json (`--max-warnings`). It fails on a *new* `any` and
 *   passes when one is removed, so the budget only ever ratchets down.
 */
module.exports = {
  root: true,
  env: { browser: true, es2020: true },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react-hooks/recommended',
  ],
  // `vite.config.ts.timestamp-*.mjs` are emitted next to the config on every build.
  ignorePatterns: ['dist', 'node_modules', '.vite', '*.timestamp-*.mjs'],
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module',
    ecmaFeatures: { jsx: true },
  },
  plugins: ['@typescript-eslint', 'react-refresh'],
  rules: {
    'react-refresh/only-export-components': [
      'error',
      {
        allowConstantExport: true,
        // Non-component exports these modules are imported for elsewhere:
        // variant helpers and the theme hook.
        allowExportNames: [
          'badgeVariants',
          'buttonVariants',
          'skeletonVariants',
          'controlClassName',
          'useTheme',
          'conditionsToRows',
          'rowsToConditions',
        ],
      },
    ],
    '@typescript-eslint/no-unused-vars': [
      'error',
      { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
    ],
    '@typescript-eslint/no-explicit-any': 'warn',
    'no-console': ['warn', { allow: ['warn', 'error'] }],
  },
  overrides: [
    {
      // Build/tooling config at the web/ root runs on Node, not in the browser.
      files: ['*.config.ts', '*.config.js', '*.cjs'],
      env: { node: true, browser: false },
      rules: { 'no-console': 'off' },
    },
  ],
}
