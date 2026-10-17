import type { CiNeeds } from './ci-pipeline.contract'
import { commitSyncStateLines, installEngineLines } from './engine-install.algorithm'
import { ENGINE_BINARY } from './engine-download.config'

const indented = (lines: string[], spaces: number): string[] => lines.map(line => `${' '.repeat(spaces)}${line}`)

/** The CircleCI configuration for these needs. */
export function renderCircleCi (needs: CiNeeds): string {
  const lines: string[] = [
    '# LoreMaster: syncs the repository documentation. CircleCI reads one file, .circleci/config.yml: if you already have',
    '# one, copy the job and the workflow into it instead of replacing the file.',
    'version: 2.1',
    '',
    'jobs:',
    '  lore-master-docs:',
    '    docker:',
    '      - image: cimg/base:stable',
    '    environment:',
    `      LORE_MASTER_VERSION: '${needs.version}'`,
    '    steps:',
    '      - checkout',
  ]

  if (needs.prepareCommands.length > 0) {
    lines.push("      # A generator's prepare command runs in this job: install what it needs first.")
    for (const command of needs.prepareCommands) {
      lines.push(`      #   ${command}`)
    }
  }

  lines.push(
    '      - run:',
    '          name: Install the LoreMaster engine',
    '          command: |',
    ...indented(installEngineLines('$LORE_MASTER_VERSION'), 12),
    '      - run:',
    '          name: Sync the documentation',
    `          command: ./${ENGINE_BINARY} ${needs.command}`,
  )
  if (needs.commitsAnnotations) {
    lines.push('      - run:', '          name: Commit the sync state', '          command: |', ...indented(commitSyncStateLines('git push origin HEAD:main'), 12))
  }

  lines.push(
    '',
    'workflows:',
    '  lore-master:',
    '    jobs:',
    '      - lore-master-docs:',
    '          filters:',
    '            branches:',
    '              only: main',
  )

  return `${lines.join('\n')}\n`
}
