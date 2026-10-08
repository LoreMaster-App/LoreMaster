import type { TreeStatus } from '../engine-protocol'
import type { PageStatus } from './page-tree.contract'

/**
 * Combines a file's own status with what a read-only plan said about its page. Without a
 * plan the local status stands. With one, a platform edit shows as remote changes, or as a
 * conflict when the file changed too; the plan's own action names (update, move,
 * rename_title) all mean the file is ahead.
 */
export function combineStatus (local: TreeStatus | undefined, remoteKind: string | undefined): PageStatus | undefined {
  if (local === undefined) {
    return undefined
  }
  switch (remoteKind) {
    case 'conflict': { return local === 'local-changes' ? 'conflict' : 'remote-changes'
    }
    case 'pull': { return 'remote-changes'
    }
    case 'adopt': { return 'adopt'
    }
    case 'create': { return 'new'
    }
    case 'update':
    case 'move':
    case 'rename_title': { return 'local-changes'
    }
    case 'unchanged': { return 'synced'
    }
    default: { return local
    }
  }
}

/** How a status is drawn: a codicon, a theme colour, and what it means in words. */
export interface StatusPresentation {
  icon:   string
  color:  string
  label:  string
  detail: string
}

const PRESENTATION: Record<PageStatus, StatusPresentation> = {
  'new':            { icon: 'diff-added', color: 'charts.green', label: 'New', detail: 'Not on the platform yet; the next sync creates it.' },
  'synced':         { icon: 'check', color: 'testing.iconPassed', label: 'Synced', detail: 'The file is the one last synced.' },
  'local-changes':  { icon: 'edit', color: 'charts.yellow', label: 'Local changes', detail: 'The file changed since the last sync; the next sync updates the page.' },
  'remote-changes': { icon: 'cloud-download', color: 'charts.blue', label: 'Remote changes', detail: 'The page was edited on the platform since the last sync.' },
  'conflict':       { icon: 'warning', color: 'list.errorForeground', label: 'Conflict', detail: 'Both the file and the page changed since the last sync.' },
  'adopt':          { icon: 'link', color: 'charts.purple', label: 'Will adopt', detail: 'A page with this title exists; the next sync takes it over.' },
}

export function statusPresentation (status: PageStatus): StatusPresentation {
  return PRESENTATION[status]
}
