import { CI_PROVIDERS, type CiProvider } from './ci-provider.config'

/**
 * The CI systems the repository already uses, from the workspace-relative files it holds. The
 * result keeps the providers' own order; none means the user has to say which CI they mean.
 */
export function detectCiProviders (files: readonly string[]): CiProvider[] {
  return CI_PROVIDERS.filter(provider => files.some(file => matchesAny(file, provider.detectGlobs)))
}

/** A glob here is a path with at most one `*` in its last segment. */
function matchesAny (file: string, globs: readonly string[]): boolean {
  return globs.some(glob => {
    const star = glob.indexOf('*')
    if (star === -1) {
      return file === glob
    }
    const prefix = glob.slice(0, star)
    const suffix = glob.slice(star + 1)

    return file.startsWith(prefix) && file.endsWith(suffix) && !file.slice(prefix.length, file.length - suffix.length).includes('/')
  })
}
