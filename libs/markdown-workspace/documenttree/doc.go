// Package documenttree decides where each document sits in the page hierarchy and
// builds the tree the sync walks parents-first. The rules, in order of precedence: an
// explicit "parent:" in the annotation, a dotted file name, the directory's index
// file, and finally the page the user selected as the sync's root.
package documenttree
