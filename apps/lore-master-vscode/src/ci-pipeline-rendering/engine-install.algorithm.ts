import { ENGINE_BINARY } from './engine-download.config'

const RELEASES = 'https://github.com/LoreMaster-App/LoreMaster/releases/download'

/**
 * The shell lines that download the engine for a version, check it against the published checksums and unpack it. The version is
 * the shell's own variable (`$LORE_MASTER_VERSION`), so a pipeline pins it in one place.
 */
export function installEngineLines (versionVariable: string): string[] {
  return [
    `BASE="${RELEASES}/lore-master-vscode%40${versionVariable}"`,
    'curl -fsSLO "$BASE/go-app-lore-master-engine-linux-amd64.zip"',
    `curl -fsSLO "$BASE/lore-master-engine-${versionVariable}-sha256.txt"`,
    `sha256sum --check --ignore-missing "lore-master-engine-${versionVariable}-sha256.txt"`,
    'unzip -o go-app-lore-master-engine-linux-amd64.zip',
    `chmod +x ${ENGINE_BINARY}`,
  ]
}

/**
 * The shell lines that commit the sync state the engine wrote (the page ids and hashes in `lore-master.json`, and any file it
 * rewrote) and push it, with a message that tells the CI not to run again for it.
 */
export function commitSyncStateLines (pushCommand: string): string[] {
  return [
    'git config user.name "lore-master"',
    'git config user.email "noreply@lore-master.invalid"',
    'git add -u -- "*.md"',
    'git add -- lore-master.json || true',
    'git diff --cached --quiet || git commit -m "docs: update LoreMaster sync state [skip ci]"',
    pushCommand,
  ]
}
