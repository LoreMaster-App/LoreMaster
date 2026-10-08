package changedetection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Take looks at the workspace. A file whose size and modification time match its entry in
// previous keeps that entry's hash without being read again, so a look at an unchanged workspace
// costs one stat per file. A file that cannot be read is left out, as if it were not there.
func Take(ctx context.Context, workspaceRoot string, previous Snapshot) (Snapshot, error) {
	snapshot := Snapshot{Files: map[string]FileState{}}
	err := filepath.WalkDir(workspaceRoot, func(full string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if full != workspaceRoot && IsSkippedFolder(entry.Name()) {
				return filepath.SkipDir
			}

			return nil
		}
		relative, err := filepath.Rel(workspaceRoot, full)
		if err != nil {
			return nil
		}
		relative = filepath.ToSlash(relative)
		if !IsWatchedFile(relative) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		state := FileState{Size: info.Size(), ModTime: info.ModTime().UnixNano()}
		if known, found := previous.Files[relative]; found && known.Size == state.Size && known.ModTime == state.ModTime {
			state.Hash = known.Hash
		} else if state.Hash, err = hashFile(full); err != nil {
			return nil
		}
		snapshot.Files[relative] = state

		return nil
	})

	return snapshot, err
}

func hashFile(full string) (string, error) {
	file, err := os.Open(full)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}
