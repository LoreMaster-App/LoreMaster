package sitepublish

import "fmt"

// ForeignFolderError means the folder to write the site into already holds files that
// LoreMaster did not put there. Nothing was changed.
type ForeignFolderError struct {
	Dir     string
	Entries int
}

func (e *ForeignFolderError) Error() string {
	return fmt.Sprintf("%s already holds %d other file(s) that LoreMaster did not write; choose an empty folder or a subfolder (a build replaces everything in its folder)", e.Dir, e.Entries)
}
