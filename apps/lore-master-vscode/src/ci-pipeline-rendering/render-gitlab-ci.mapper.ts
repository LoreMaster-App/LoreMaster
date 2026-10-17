import type { CiNeeds } from './ci-pipeline.contract'
import { commitSyncStateLines, installEngineLines } from './engine-install.algorithm'
import { ENGINE_BINARY } from './engine-download.config'

/** What the job runs to push: the remote needs a token with write access, because the job's own is read-only. */
const GIT_REMOTE = 'git remote set-url origin "https://oauth2:${LORE_MASTER_GIT_TOKEN}@${CI_SERVER_HOST}/${CI_PROJECT_PATH}.git"'

const indented = (lines: string[], spaces: number): string[] => lines.map(line => `${' '.repeat(spaces)}${line}`)

/** The GitLab CI/CD file for these needs: its own file, which `.gitlab-ci.yml` includes. */
export function renderGitLabCi (needs: CiNeeds): string {
  const writes = needs.pushesToGit || needs.commitsAnnotations
  const lines: string[] = [
    '# LoreMaster: syncs the repository documentation. This file is its own: add it to your .gitlab-ci.yml with',
    '#',
    '#   include:',
    '#     - local: .gitlab/lore-master.yml',
    '#',
    'lore-master-docs:',
    '  image: ubuntu:24.04',
    '  rules:',
    "    - if: '$CI_PIPELINE_SOURCE == \"push\" && $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH'",
    '      changes:',
    "        - '**/*.md'",
    '        - .vscode/settings.json',
    '  variables:',
    `    LORE_MASTER_VERSION: '${needs.version}'`,
    '  before_script:',
    '    - apt-get update -qq && apt-get install -y -qq curl unzip git ca-certificates',
  ]

  if (needs.prepareCommands.length > 0) {
    lines.push("    # A generator's prepare command runs in this job: install what it needs here first.")
    for (const command of needs.prepareCommands) {
      lines.push(`    #   ${command}`)
    }
  }

  lines.push('  script:', '    - |', ...indented(installEngineLines('$LORE_MASTER_VERSION'), 6))
  if (writes) {
    lines.push(`    - ${GIT_REMOTE}`)
  }
  lines.push(`    - ./${ENGINE_BINARY} ${needs.command}`)
  if (needs.commitsAnnotations) {
    lines.push('    - |', ...indented(commitSyncStateLines('git push origin HEAD:$CI_COMMIT_BRANCH'), 6))
  }

  return `${lines.join('\n')}\n`
}
