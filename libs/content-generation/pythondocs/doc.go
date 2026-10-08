// Package pythondocs is the python-docs generator: it runs pydoc-markdown, which reads the
// source statically (nothing is imported or executed), over the Python projects of the
// workspace and writes a page per module under the output folder, nested like the packages.
// pydoc-markdown is expected on the PATH or in a .venv; when it is not there the generator
// says what to install.
package pythondocs
