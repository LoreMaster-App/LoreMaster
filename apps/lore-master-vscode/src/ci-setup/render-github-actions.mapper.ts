import { ENGINE_BINARY } from './ci-provider.config'
import type { CiNeeds } from './ci-pipeline.contract'

const RELEASES = 'https://github.com/LoreMaster-App/LoreMaster/releases/download'

/** The workflow file for these needs. */
export function renderGitHubActions (needs: CiNeeds): string {
  const writes = needs.pushesToGit || needs.commitsAnnotations
  const lines: string[] = [
    'name: LoreMaster',
    '',
    'on:',
    '  push:',
    '    branches: [main]',
    "    paths: ['**/*.md', '.vscode/settings.json']",
    '  workflow_dispatch:',
    '',
    'concurrency:',
    '  group: lore-master',
    '  cancel-in-progress: false',
    '',
    'permissions:',
    `  contents: ${writes ? 'write' : 'read'}`,
    '',
    'jobs:',
    '  docs:',
    '    runs-on: ubuntu-latest',
    '    steps:',
    '      - uses: actions/checkout@v7',
    '',
  ]

  if (needs.prepareCommands.length > 0) {
    lines.push("      # A generator's prepare command runs in this job: install what it needs first.")
    for (const command of needs.prepareCommands) {
      lines.push(`      #   ${command}`)
    }
    lines.push('')
  }

  lines.push(
    '      - name: Install the LoreMaster engine',
    '        env:',
    `          VERSION: ${needs.version}`,
    '        run: |',
    `          BASE="${RELEASES}/lore-master-vscode%40$VERSION"`,
    '          curl -fsSLO "$BASE/go-app-lore-master-engine-linux-amd64.zip"',
    '          curl -fsSLO "$BASE/lore-master-engine-$VERSION-sha256.txt"',
    '          sha256sum --check --ignore-missing "lore-master-engine-$VERSION-sha256.txt"',
    '          unzip -o go-app-lore-master-engine-linux-amd64.zip',
    `          chmod +x ${ENGINE_BINARY}`,
    '',
    '      - name: Sync the documentation',
  )

  if (needs.secrets.length > 0) {
    lines.push('        env:')
    for (const secret of needs.secrets) {
      lines.push(`          ${secret.name}: \${{ secrets.${secret.name} }}`)
    }
  }
  lines.push(`        run: ./${ENGINE_BINARY} ${needs.command}`)

  if (needs.commitsAnnotations) {
    lines.push(
      '',
      '      - name: Commit the page annotations',
      '        run: |',
      '          git config user.name "lore-master[bot]"',
      '          git config user.email "noreply@users.noreply.github.com"',
      '          git add -u -- "*.md"',
      '          git diff --cached --quiet || git commit -m "docs: update LoreMaster page annotations [skip ci]"',
      '          git push',
    )
  }

  return `${lines.join('\n')}\n`
}
