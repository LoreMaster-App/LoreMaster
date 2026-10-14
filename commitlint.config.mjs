import conventional from '@commitlint/config-conventional'

// `version(<project>)[<version>]: <message>` forces a release version (#283), so its types and its
// `[<version>]` part are accepted alongside the conventional ones.
export default {
  extends:      ['@commitlint/config-conventional'],
  parserPreset: {
    parserOpts: {
      headerPattern:        /^(\w*)(?:\((.*)\))?(?:\[(.*)\])?!?: (.*)$/,
      headerCorrespondence: ['type', 'scope', 'version', 'subject'],
    },
  },
  rules: {
    'type-enum': [2, 'always', [...conventional.rules['type-enum'][2], 'mnci-version', 'mnci-ver', 'mnci-force', 'mnci-v', 'version', 'ver', 'force', 'v']],
  },
}
