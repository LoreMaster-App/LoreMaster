package syncplanning

import (
	"sync"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documenttree"
)

// prefetchWorkers is how many page lookups are in flight at once: enough to hide a slow server's
// round trips, few enough not to hammer it.
const prefetchWorkers = 8

// remoteLookup is what the platform said about one document: its page when it has an annotation
// for this output, otherwise the pages already holding its title.
type remoteLookup struct {
	page  platformport.RemotePage
	found []platformport.RemotePage
	err   error
}

// prefetch looks up every document's remote state concurrently, so the walk that follows reads
// the answers instead of waiting on one request per file. Errors are kept per document and
// surface when the walk reaches it, so the first failure in tree order wins.
func (run *planner) prefetch() {
	run.lookups = map[documentdiscovery.DocumentPath]*remoteLookup{}
	run.input.Tree.Walk(func(node *documenttree.TreeNode, _ int) {
		run.lookups[node.Document.Path] = &remoteLookup{}
	})

	var group sync.WaitGroup
	slots := make(chan struct{}, prefetchWorkers)
	run.input.Tree.Walk(func(node *documenttree.TreeNode, _ int) {
		group.Add(1)
		slots <- struct{}{}
		go func() {
			defer group.Done()
			defer func() { <-slots }()
			lookup := run.lookups[node.Document.Path]
			if annotation := node.Document.Annotation; annotation == nil || annotation.PageID == "" || !sameTarget(annotation, run.input.Output) {
				lookup.found, lookup.err = run.platform.FindPagesByTitle(run.ctx, run.space, run.titles[node.Document.Path])

				return
			}
			lookup.page, lookup.err = run.platform.GetPage(run.ctx, node.Document.Annotation.PageID)
		}()
	})
	group.Wait()
}
