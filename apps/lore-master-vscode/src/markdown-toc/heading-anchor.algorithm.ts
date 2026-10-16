import type { DocumentHeading } from './find-headings.algorithm'

/**
 * A heading's text as a reader sees it: links keep their text, images their alt text, and emphasis, code and
 * strikethrough marks, HTML tags and escapes are gone.
 */
export function plainHeadingText (text: string): string {
  return text
    .replaceAll(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replaceAll(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replaceAll(/<[^>]+>/g, '')
    .replaceAll(/[*`~]/g, '')
    // An underscore that opens or closes emphasis goes; one inside a word (`snake_case`) stays.
    .replaceAll(/(^|[^\p{L}\p{N}_])_+(?=[\p{L}\p{N}])/gu, '$1')
    .replaceAll(/(?<=[\p{L}\p{N}])_+(?=$|[^\p{L}\p{N}_])/gu, '')
    .replaceAll(/\\(.)/g, '$1')
    .trim()
}

/** GitHub's anchor for a heading text: lower case, punctuation gone except `-` and `_`, spaces made hyphens. */
export function githubSlug (text: string): string {
  return plainHeadingText(text)
    .toLowerCase()
    .replaceAll(/[^\p{L}\p{M}\p{N}\p{Pc}\- ]/gu, '')
    .replaceAll(' ', '-')
}

/** The anchor of every heading, in order; a repeated one gets `-1`, `-2` and so on, as GitHub numbers them. */
export function anchorsOf (headings: readonly DocumentHeading[]): string[] {
  const seen = new Map<string, number>()

  return headings.map(heading => {
    const slug = githubSlug(heading.text)
    const count = seen.get(slug)
    seen.set(slug, (count ?? 0) + 1)

    return count === undefined ? slug : `${slug}-${count}`
  })
}
