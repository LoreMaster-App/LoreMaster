package documentexport

// File is one file an export produced.
type File struct {
	// Name is the file name, without a folder.
	Name    string
	Content []byte
}
