import { ENGINE_BINARY } from './ci-provider.config'
import type { CiNeeds } from './ci-pipeline.contract'

const RELEASES = 'https://github.com/LoreMaster-App/LoreMaster/releases/download'

/** The pipeline file for these needs. */
export function renderAzurePipelines (needs: CiNeeds): string {
  const writes = needs.pushesToGit || needs.commitsAnnotations
  const lines: string[] = [
    '# LoreMaster: syncs the repository documentation. Register it in Azure DevOps with',
    '# Pipelines > New pipeline > Existing Azure Pipelines YAML file.',
    'trigger:',
    '  branches:',
    '    include: [main]',
    '  paths:',
    '    include:',
    "      - '**/*.md'",
    "      - '.vscode/settings.json'",
    '',
    'pr: none',
    '',
    'pool:',
    '  vmImage: ubuntu-latest',
    '',
    'variables:',
    `  loreMasterVersion: '${needs.version}'`,
    '',
    'steps:',
    '  - checkout: self',
    `    persistCredentials: ${writes ? 'true' : 'false'}`,
    '    clean: true',
    '',
  ]

  if (needs.prepareCommands.length > 0) {
    lines.push("  # A generator's prepare command runs in this job: install what it needs first.")
    for (const command of needs.prepareCommands) {
      lines.push(`  #   ${command}`)
    }
    lines.push('')
  }

  lines.push(
    '  - bash: |',
    `      BASE="${RELEASES}/lore-master-vscode%40$(loreMasterVersion)"`,
    '      curl -fsSLO "$BASE/go-app-lore-master-engine-linux-amd64.zip"',
    '      curl -fsSLO "$BASE/lore-master-engine-$(loreMasterVersion)-sha256.txt"',
    '      sha256sum --check --ignore-missing "lore-master-engine-$(loreMasterVersion)-sha256.txt"',
    '      unzip -o go-app-lore-master-engine-linux-amd64.zip',
    `      chmod +x ${ENGINE_BINARY}`,
    '    displayName: Install the LoreMaster engine',
    '',
    '  - bash: |',
    `      ./${ENGINE_BINARY} ${needs.command}`,
    '    displayName: Sync the documentation',
  )

  if (needs.secrets.length > 0) {
    lines.push('    env:')
    for (const secret of needs.secrets) {
      lines.push(`      ${secret.name}: $(${secret.name})`)
    }
  }

  if (needs.commitsAnnotations) {
    lines.push(
      '',
      '  - bash: |',
      '      git config user.name "lore-master"',
      '      git config user.email "noreply@lore-master.invalid"',
      '      git add -u -- "*.md"',
      '      git add -- lore-master.json || true',
      '      git diff --cached --quiet || git commit -m "docs: update LoreMaster sync state [skip ci]"',
      '      git push origin HEAD:$(Build.SourceBranchName)',
      '    displayName: Commit the sync state',
    )
  }

  return `${lines.join('\n')}\n`
}
