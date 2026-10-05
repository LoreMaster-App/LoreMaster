package sitepublish

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// gitClient runs the git command line. The engine shells out to the user's own git so a
// publish uses the credentials and remotes they already have — the chosen auth model.
type gitClient struct {
	// binary is the git executable; "git" by default.
	binary string
}

func newGitClient() gitClient { return gitClient{binary: "git"} }

// run executes git in dir (the current directory when dir is empty) with args, returning
// trimmed stdout. On failure the error carries git's stderr so the caller can report why a
// publish could not proceed.
func (g gitClient) run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, g.binary, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}
