import type { CiProvider } from './ci-provider.config'
import type { CiNeeds, CiPipelineFile } from '../ci-pipeline-rendering'

const WHERE_TO_ADD_SECRETS: Record<CiProvider['id'], string> = {
  'github-actions':  'On GitHub: **Settings > Secrets and variables > Actions > New repository secret**.',
  'azure-pipelines': 'In Azure DevOps: open the pipeline, **Edit > Variables > New variable**, and tick **Keep this value secret**.',
}

const FENCE_LANGUAGE: Record<CiProvider['id'], string> = {
  'github-actions':  'yaml',
  'azure-pipelines': 'yaml',
}

/** The Markdown the session opens: what to create, what to commit and what the pipeline does. */
export function renderCiGuide (provider: CiProvider, needs: CiNeeds, pipeline: CiPipelineFile): string {
  const sections: string[] = [
    `# Run LoreMaster in ${provider.label}`,
    '',
    `This guide was written from the settings of this workspace. The pipeline syncs your documentation on every push to \`main\` that touches Markdown, with the LoreMaster engine **${needs.version}** (the version of your extension), so CI does what the editor does.`,
  ]

  if (needs.secrets.length > 0) {
    sections.push('', '## 1. Create the secrets', '', WHERE_TO_ADD_SECRETS[provider.id], '', '| Name | What it is |', '|---|---|')
    for (const secret of needs.secrets) {
      sections.push(`| \`${secret.name}\` | ${secret.purpose} |`)
    }
    sections.push(
      '',
      'Credentials never go into settings or into the repository. For Confluence Server before 7.9 use `LORE_MASTER_USER` and `LORE_MASTER_PASSWORD` instead of the token.',
    )
  } else {
    sections.push('', '## 1. Secrets', '', 'Your outputs publish through git, which carries its own credentials. There is nothing to create.')
  }

  sections.push('', '## 2. Commit your settings', '')
  if (needs.settingsInRepo) {
    sections.push('The outputs and generators are in the repository\'s `.vscode/settings.json`, so CI reads the same configuration as the editor.')
  } else {
    sections.push(
      'Your outputs are not in the repository\'s `.vscode/settings.json` (they are in your user or workspace-file settings), and CI has neither. Add them to `.vscode/settings.json` and commit it:',
      '',
      '```json',
      needs.settingsJson,
      '```',
    )
  }

  sections.push(
    '',
    '## 3. Add the pipeline',
    '',
    `Save this as \`${pipeline.path}\`:`,
    '',
    `\`\`\`${FENCE_LANGUAGE[provider.id]}`,
    pipeline.content.trimEnd(),
    '```',
  )

  const notes: string[] = []
  if (needs.pushesToGit) {
    notes.push('The job pushes the site to the output\'s branch, so it needs permission to write to the repository (the file asks for `contents: write`).')
  }
  if (needs.commitsAnnotations) {
    notes.push('The sync records each page (id, version, hashes) in `lore-master.json` at the workspace root. The job commits that file back with `[skip ci]`, so the next run updates the pages instead of adopting them again. The pipeline identity needs permission to push to `main`; if the branch is protected, allow it or push to a branch and merge.')
  }
  if (needs.prepareCommands.length > 0) {
    notes.push(`Generators run \`prepare\` commands in the job (${needs.prepareCommands.map(command => `\`${command}\``).join(', ')}). Install their toolchains in a step before the sync.`)
  }
  notes.push('Try it first without changing anything: `lore-master-engine sync --dry-run` shows the plan and writes nothing.', 'When you upgrade the extension, change the pinned version in the pipeline too; the command line follows the extension\'s versions.')

  sections.push('', '## Good to know', '', ...notes.map(note => `- ${note}`))

  return `${sections.join('\n')}\n`
}
