import type { CiNeeds } from './ci-pipeline.contract'
import { commitSyncStateLines, installEngineLines } from './engine-install.algorithm'
import { ENGINE_BINARY } from './engine-download.config'

const indented = (lines: string[], spaces: number): string[] => lines.map(line => `${' '.repeat(spaces)}${line}`)

/** The Bitbucket Pipelines file for these needs. */
export function renderBitbucketPipelines (needs: CiNeeds): string {
  const lines: string[] = [
    '# LoreMaster: syncs the repository documentation. Bitbucket reads one file, bitbucket-pipelines.yml: if you already',
    '# have pipelines, copy this step into yours instead of replacing the file.',
    'image: atlassian/default-image:4',
    '',
    'pipelines:',
    '  branches:',
    '    main:',
    '      - step:',
    '          name: Sync the documentation with LoreMaster',
    '          condition:',
    '            changesets:',
    '              includePaths:',
    "                - '**/*.md'",
    '                - .vscode/settings.json',
    '          script:',
  ]

  if (needs.prepareCommands.length > 0) {
    lines.push("            # A generator's prepare command runs in this step: install what it needs first.")
    for (const command of needs.prepareCommands) {
      lines.push(`            #   ${command}`)
    }
  }

  lines.push(
    '            - |',
    `              export LORE_MASTER_VERSION='${needs.version}'`,
    ...indented(installEngineLines('$LORE_MASTER_VERSION'), 14),
    `            - ./${ENGINE_BINARY} ${needs.command}`,
  )
  if (needs.commitsAnnotations) {
    lines.push('            - |', ...indented(commitSyncStateLines('git push origin HEAD:main'), 14))
  }

  return `${lines.join('\n')}\n`
}
