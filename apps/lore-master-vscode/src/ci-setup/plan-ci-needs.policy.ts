import { isSiteOutput, platformOf } from '../config-description'
import { authMethodsFor } from '../connection-setup'
import type { ConfigDescription, ConnectionDescriptor, CredentialKindDescriptor, Output, Settings } from '../engine-protocol'
import type { CiNeeds, CiSecret } from '../ci-pipeline-rendering'

export interface PlanCiNeedsInput {
  settings:       Settings
  /** The engine's description of the platforms: which sign in, and which publish with git. */
  description:    ConfigDescription
  version:        string
  settingsInRepo: boolean
}

/**
 * Decides what the pipeline must do for these settings: the one sync command, the credentials
 * each Confluence output needs, and the side effects the job has to be allowed to have.
 * Credentials are never part of the settings, so they are only named here.
 */
export function planCiNeeds ({ settings, description, version, settingsInRepo }: PlanCiNeedsInput): CiNeeds {
  const outputs = settings.outputs
  const generators = settings.generators ?? []
  const prepareCommands = generators.flatMap(generator => generator.prepare ? [generator.prepare] : [])
  const confluence = outputs.filter(output => platformOf(description, output)?.connection !== undefined)

  const flags = [
    ...generators.length > 0 ? ['--generate'] : [],
    ...prepareCommands.length > 0 ? ['--prepare'] : [],
    '--yes',
  ]

  return {
    version,
    command:            `sync ${flags.join(' ')}`,
    secrets:            secretsFor(confluence, description),
    pushesToGit:        outputs.some(output => isSiteOutput(description, output)),
    commitsAnnotations: confluence.length > 0,
    prepareCommands,
    settingsInRepo,
    settingsJson:       settingsJson(settings),
  }
}

/**
 * The credentials the outputs that sign in need, named by the environment variables the engine's description gives each field of
 * the sign-in method. The method is the first one the output's edition accepts; the edition comes from the description's hints
 * about the address (a site of no hinted edition takes the first method meant for the others).
 */
function secretsFor (outputs: readonly Output[], description: ConfigDescription): CiSecret[] {
  const wanted = outputs.flatMap(output => {
    const platform = platformOf(description, output)
    const connection = platform?.connection
    const method = connection ? preferredMethod(connection, output.baseUrl) : undefined

    return platform && method ? method.fields.map(field => ({ name: field.env, purpose: `${field.help ?? field.label} (${platform.label})` })) : []
  })

  return wanted.filter((secret, index) => wanted.findIndex(other => other.name === secret.name) === index)
}

/** The sign-in method a pipeline uses for a site: the first one its edition accepts. */
function preferredMethod (connection: ConnectionDescriptor, baseUrl: string): CredentialKindDescriptor | undefined {
  const hints = connection.editionHints ?? []
  const host = hostOf(baseUrl)
  const hinted = hints.find(hint => host.endsWith(hint.hostSuffix))
  if (hinted) {
    return authMethodsFor(connection, hinted.edition)[0]
  }
  const hintedEditions = new Set(hints.map(hint => hint.edition))

  return connection.kinds.find(kind => kind.editions === undefined || kind.editions.length === 0 || kind.editions.some(edition => !hintedEditions.has(edition)))
}

function hostOf (baseUrl: string): string {
  try {
    return new URL(baseUrl).hostname
  } catch {
    return ''
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
