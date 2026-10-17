import { isGitHubOutput, type Output, type Settings } from '../engine-protocol'
import type { CiNeeds, CiSecret } from '../ci-pipeline-rendering'

export interface PlanCiNeedsInput {
  settings:       Settings
  version:        string
  settingsInRepo: boolean
}

const CLOUD_SECRETS: CiSecret[] = [
  { name: 'LORE_MASTER_EMAIL', purpose: 'The Atlassian account email of the API token (Confluence Cloud).' },
  { name: 'LORE_MASTER_TOKEN', purpose: 'An Atlassian API token for that account (Confluence Cloud).' },
]

const SELF_HOSTED_SECRETS: CiSecret[] = [
  { name: 'LORE_MASTER_PAT', purpose: 'A personal access token (Confluence Data Center, or Server 7.9 and later).' },
]

/**
 * Decides what the pipeline must do for these settings: the one sync command, the credentials
 * each Confluence output needs, and the side effects the job has to be allowed to have.
 * Credentials are never part of the settings, so they are only named here.
 */
export function planCiNeeds ({ settings, version, settingsInRepo }: PlanCiNeedsInput): CiNeeds {
  const outputs = settings.outputs
  const generators = settings.generators ?? []
  const prepareCommands = generators.flatMap(generator => generator.prepare ? [generator.prepare] : [])
  const confluence = outputs.filter(output => !isGitHubOutput(output))

  const flags = [
    ...generators.length > 0 ? ['--generate'] : [],
    ...prepareCommands.length > 0 ? ['--prepare'] : [],
    '--yes',
  ]

  return {
    version,
    command:            `sync ${flags.join(' ')}`,
    secrets:            secretsFor(confluence),
    pushesToGit:        outputs.some(output => isGitHubOutput(output)),
    commitsAnnotations: confluence.length > 0,
    prepareCommands,
    settingsInRepo,
    settingsJson:       settingsJson(settings),
  }
}

/** Cloud sites sit on atlassian.net and use an email with a token; anything else, a token alone. */
function secretsFor (confluence: readonly Output[]): CiSecret[] {
  const wanted = confluence.flatMap(output => isCloudSite(output.baseUrl) ? CLOUD_SECRETS : SELF_HOSTED_SECRETS)

  return wanted.filter((secret, index) => wanted.findIndex(other => other.name === secret.name) === index)
}

function isCloudSite (baseUrl: string): boolean {
  try {
    return new URL(baseUrl).hostname.endsWith('.atlassian.net')
  } catch {
    return false
  }
}

/** The settings as the `loreMaster.*` keys a settings file or `--settings` takes. */
function settingsJson (settings: Settings): string {
  const keys: Record<string, unknown> = { 'loreMaster.outputs': settings.outputs }
  if (settings.generators && settings.generators.length > 0) {
    keys['loreMaster.generators'] = settings.generators
  }
  if (settings.ignore && settings.ignore.length > 0) {
    keys['loreMaster.ignore'] = settings.ignore
  }
  if (settings.skipGitignored !== undefined) {
    keys['loreMaster.skipGitignored'] = settings.skipGitignored
  }

  return JSON.stringify(keys, undefined, 2)
}
