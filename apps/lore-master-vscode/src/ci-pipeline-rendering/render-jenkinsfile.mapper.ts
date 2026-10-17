import type { CiNeeds } from './ci-pipeline.contract'
import { commitSyncStateLines, installEngineLines } from './engine-install.algorithm'
import { ENGINE_BINARY } from './engine-download.config'

const indented = (lines: string[], spaces: number): string[] => lines.map(line => `${' '.repeat(spaces)}${line}`)

/** The Jenkins declarative pipeline for these needs. */
export function renderJenkinsfile (needs: CiNeeds): string {
  const lines: string[] = [
    '// LoreMaster: syncs the repository documentation. Make a Pipeline (or Multibranch Pipeline) job from this',
    '// repository and set its Script Path to Jenkinsfile.lore-master.',
    'pipeline {',
    '  agent any',
    '  options { disableConcurrentBuilds() }',
    '  environment {',
    `    LORE_MASTER_VERSION = '${needs.version}'`,
    '  }',
    '  stages {',
    "    stage('Sync the documentation') {",
    '      when {',
    '        allOf {',
    "          expression { env.BRANCH_NAME == null || env.BRANCH_NAME == 'main' }",
    '          anyOf {',
    "            changeset '**/*.md'",
    "            changeset '.vscode/settings.json'",
    '          }',
    '        }',
    '      }',
  ]

  if (needs.secrets.length > 0) {
    lines.push('      // Each credential is a "Secret text" credential in Jenkins, with the name as its ID.', '      environment {')
    for (const secret of needs.secrets) {
      lines.push(`        ${secret.name} = credentials('${secret.name}')`)
    }
    lines.push('      }')
  }

  lines.push('      steps {')
  if (needs.prepareCommands.length > 0) {
    lines.push("        // A generator's prepare command runs in this stage: install what it needs first.")
    for (const command of needs.prepareCommands) {
      lines.push(`        //   ${command}`)
    }
  }
  lines.push("        sh '''", ...indented(installEngineLines('$LORE_MASTER_VERSION'), 10), "        '''", `        sh './${ENGINE_BINARY} ${needs.command}'`)
  if (needs.commitsAnnotations) {
    lines.push("        sh '''", ...indented(commitSyncStateLines('git push origin HEAD:main'), 10), "        '''")
  }
  lines.push('      }', '    }', '  }', '}')

  return `${lines.join('\n')}\n`
}
