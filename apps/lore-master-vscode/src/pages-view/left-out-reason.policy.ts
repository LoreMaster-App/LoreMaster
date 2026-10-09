import type { LeftOutFile } from '../engine-protocol'

/** How a file the scan left out is explained to the user: a short reason beside its name and
 *  the longer sentence for its tooltip. */
export interface LeftOutReason {
  description: string
  detail:      string
}

/** Says which setting left the file out, and which line of it, so it can be changed. */
export function leftOutReason (file: LeftOutFile): LeftOutReason {
  switch (file.rule) {
    case 'ignore': {
      return {
        description: `ignore: ${file.pattern ?? ''}`,
        detail:      `Matched \`${file.pattern ?? ''}\` in the "ignore" list of .lore-master.yaml, which every storage leaves out.`,
      }
    }
    case 'excludes': {
      return {
        description: `excludes: ${file.pattern ?? ''}`,
        detail:      `Matched \`${file.pattern ?? ''}\` in this storage's "excludes".`,
      }
    }
    case 'gitignore': {
      return {
        description: `.gitignore: ${file.pattern ?? ''}`,
        detail:      `Matched \`${file.pattern ?? ''}\` in ${file.source ?? '.gitignore'}. Set "skipGitignored: false" to sync it anyway.`,
      }
    }
    default: {
      return {
        description: 'outside the folders to sync',
        detail:      'No "roots" entry of this storage contains the file. Add its folder to the roots to sync it.',
      }
    }
  }
}
