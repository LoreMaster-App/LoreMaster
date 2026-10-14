/**
 * The prepare commands the user approved in this session, so an unchanged command is not asked
 * about again. Kept in memory only: a new session asks again, and so does an edited command.
 */
export const approvedPrepareCommands = new Set<string>()
