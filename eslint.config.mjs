// This file is YOURS. mnci writes it once and never touches it again, so
// anything you add here survives `mnci upgrade`.
//
// The rules live in ./eslint.config.mnci.mjs, which mnci DOES rewrite on every
// upgrade — so put your changes here, not there.
import mnci from './eslint.config.mnci.mjs'

// TO CONFIGURE the shared rules, pass options to mnci() below — e.g.
// `...mnci({ verticalSlices: ['packages/*/src/**/*.ts'] })`.
//
// TO OVERRIDE a rule, append a block AFTER the spread — later blocks win, so
// one of your own beats anything above it. Give it a name, so
// `npx eslint --inspect-config` shows where the change came from:
//
//   {
//     name: 'local/legacy-app-allows-any',
//     files: ['apps/legacy/**/*.ts'],
//     rules: { '@typescript-eslint/no-explicit-any': 'off' }
//   }
//
// Do NOT edit @mnci/eslint-config inside node_modules, and do not fork it: it
// is a dependency, so `npm update` brings rule fixes in the way it brings any
// other. An override here survives that; an edit to the package does not.
export default [
  ...mnci(),
]
