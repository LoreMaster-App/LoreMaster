package changedetection

// Snapshot is one look at a workspace: what is known of each watched file, by workspace-relative
// '/'-separated path.
type Snapshot struct {
	Files map[string]FileState
}

// FileState is what is known of one file.
type FileState struct {
	Size    int64
	ModTime int64
	// Hash is the SHA-256 of the content, hex.
	Hash string
}
