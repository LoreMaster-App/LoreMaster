/** The CI systems the session writes instructions and a pipeline file for. */
export type CiProviderId = 'github-actions' | 'azure-pipelines'

/** One credential the pipeline needs, named the way the engine reads it from the environment. */
export interface CiSecret {
  name:    string
  purpose: string
}

/** What the workspace's settings say the pipeline must do, before it is spelled for a CI. */
export interface CiNeeds {
  /** The extension's version; the engine download is pinned to it. */
  version:            string
  /** The engine's command line after the binary name. */
  command:            string
  /** Credentials to create as secrets of the CI, by name. */
  secrets:            CiSecret[]
  /** Pages and wiki outputs are pushed with git, so the job needs write access to the repository. */
  pushesToGit:        boolean
  /** Confluence syncs record each page in lore-master.json; CI has to commit that file back. */
  commitsAnnotations: boolean
  /** Generators whose `prepare` command runs in the job, with their toolchains. */
  prepareCommands:    string[]
  /** False when the outputs live in user or workspace-file settings, which CI does not have. */
  settingsInRepo:     boolean
  /** The `loreMaster.*` settings as JSON, to commit when settingsInRepo is false. */
  settingsJson:       string
}

/** A rendered pipeline file and where it goes. */
export interface CiPipelineFile {
  /** Workspace-relative, '/'-separated. */
  path:    string
  content: string
}
