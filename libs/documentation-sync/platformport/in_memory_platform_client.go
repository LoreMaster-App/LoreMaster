package platformport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// InMemoryPlatform is a DocumentationPlatform held in memory, for the sync's own tests:
// a first sync, a remote edit, a conflict or an orphan can be staged without a network.
// It keeps the rules the sync relies on: titles unique per space ignoring case,
// versions that must match, idempotent marks, attachments skipped when unchanged, and
// a trash that forgets. Every call is logged in order.
type InMemoryPlatform struct {
	mu          sync.Mutex
	spaces      []Space
	pages       map[string]*StoredPage
	nextID      int
	calls       []string
	attachments map[string]map[string]UploadedFile
	// attachmentContent holds the bytes uploaded, so a two-way pull can download them back.
	attachmentContent map[string]map[string][]byte
}

// StoredPage is a page as the fake keeps it.
type StoredPage struct {
	RemotePage
	SpaceKey   string
	Body       Document
	Marked     bool
	SourcePath string
}

var _ DocumentationPlatform = (*InMemoryPlatform)(nil)

// NewInMemoryPlatform returns an empty platform with the given spaces.
func NewInMemoryPlatform(spaces ...Space) *InMemoryPlatform {
	return &InMemoryPlatform{spaces: spaces, pages: map[string]*StoredPage{}, attachments: map[string]map[string]UploadedFile{}, attachmentContent: map[string]map[string][]byte{}}
}

// SeedPage puts a page on the platform as if someone had made it, and returns its id.
func (p *InMemoryPlatform) SeedPage(spaceKey string, parentID string, title string) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.store(spaceKey, parentID, title, Document{}).ID
}

// EditRemotely bumps a page's version, as a person editing it on the platform would.
func (p *InMemoryPlatform) EditRemotely(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pages[id].Version++
}

// Page returns a stored page for inspection, or nil.
func (p *InMemoryPlatform) Page(id string) *StoredPage {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.pages[id]
}

// Calls returns the calls made so far, oldest first, as "Method arg".
func (p *InMemoryPlatform) Calls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return slices.Clone(p.calls)
}

func (p *InMemoryPlatform) log(format string, args ...any) {
	p.calls = append(p.calls, fmt.Sprintf(format, args...))
}

func (p *InMemoryPlatform) store(spaceKey string, parentID string, title string, body Document) *StoredPage {
	p.nextID++
	id := "p" + strconv.Itoa(p.nextID)
	page := &StoredPage{
		RemotePage: RemotePage{ID: id, Title: title, ParentID: parentID, Version: 1, URL: "https://docs.example/pages/" + id},
		SpaceKey:   spaceKey, Body: body,
	}
	p.pages[id] = page

	return page
}

func (p *InMemoryPlatform) titleTaken(spaceKey string, title string, except string) bool {
	for id, page := range p.pages {
		if id != except && page.SpaceKey == spaceKey && strings.EqualFold(page.Title, title) {
			return true
		}
	}

	return false
}

func (p *InMemoryPlatform) sortedIDs() []string {
	ids := make([]string, 0, len(p.pages))
	for id := range p.pages {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int {
		ai, _ := strconv.Atoi(a[1:])
		bi, _ := strconv.Atoi(b[1:])

		return ai - bi
	})

	return ids
}

// ListSpaces implements DocumentationPlatform.
func (p *InMemoryPlatform) ListSpaces(context.Context) ([]Space, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("ListSpaces")

	return slices.Clone(p.spaces), nil
}

// GetPage implements DocumentationPlatform.
func (p *InMemoryPlatform) GetPage(_ context.Context, id string) (RemotePage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("GetPage %s", id)
	page, ok := p.pages[id]
	if !ok {
		return RemotePage{}, &PageNotFoundError{ID: id}
	}

	return page.RemotePage, nil
}

// GetPageContent implements DocumentationPlatform. The fake stores bodies as Documents, so no
// conversion is needed and nothing is ever flagged.
func (p *InMemoryPlatform) GetPageContent(_ context.Context, id string) (PageContent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("GetPageContent %s", id)
	page, ok := p.pages[id]
	if !ok {
		return PageContent{}, &PageNotFoundError{ID: id}
	}

	return PageContent{Version: page.Version, Body: page.Body}, nil
}

// ListChildren implements DocumentationPlatform.
func (p *InMemoryPlatform) ListChildren(_ context.Context, parentID string) ([]RemotePage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("ListChildren %s", parentID)
	if _, exists := p.pages[parentID]; !exists && parentID != "" {
		return nil, &PageNotFoundError{ID: parentID}
	}
	var children []RemotePage
	for _, id := range p.sortedIDs() {
		if p.pages[id].ParentID == parentID {
			children = append(children, p.pages[id].RemotePage)
		}
	}

	return children, nil
}

// FindPagesByTitle implements DocumentationPlatform.
func (p *InMemoryPlatform) FindPagesByTitle(_ context.Context, space SpaceRef, title string) ([]RemotePage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("FindPagesByTitle %s %s", space.Key, title)
	var found []RemotePage
	for _, id := range p.sortedIDs() {
		if page := p.pages[id]; page.SpaceKey == space.Key && strings.EqualFold(page.Title, title) {
			found = append(found, page.RemotePage)
		}
	}

	return found, nil
}

// FindPages implements DocumentationPlatform.
func (p *InMemoryPlatform) FindPages(_ context.Context, space SpaceRef, query string, limit int) ([]RemotePage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("FindPages %s %s", space.Key, query)
	needle := strings.ToLower(strings.TrimSpace(query))
	var found []RemotePage
	for _, id := range p.sortedIDs() {
		if page := p.pages[id]; page.SpaceKey == space.Key && strings.Contains(strings.ToLower(page.Title), needle) {
			found = append(found, page.RemotePage)
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].Title < found[j].Title })
	if limit > 0 && len(found) > limit {
		found = found[:limit]
	}

	return found, nil
}

// CreatePage implements DocumentationPlatform.
func (p *InMemoryPlatform) CreatePage(_ context.Context, page NewPage) (RemotePage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("CreatePage %s", page.Title)
	if p.titleTaken(page.Space.Key, page.Title, "") {
		return RemotePage{}, &TitleTakenError{Title: page.Title}
	}
	if page.ParentID != "" {
		if _, ok := p.pages[page.ParentID]; !ok {
			return RemotePage{}, &PageNotFoundError{ID: page.ParentID}
		}
	}

	return p.store(page.Space.Key, page.ParentID, page.Title, page.Body).RemotePage, nil
}

// UpdatePage implements DocumentationPlatform.
func (p *InMemoryPlatform) UpdatePage(_ context.Context, update PageUpdate) (RemotePage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("UpdatePage %s v%d", update.ID, update.ExpectedVersion)
	page, ok := p.pages[update.ID]
	if !ok {
		return RemotePage{}, &PageNotFoundError{ID: update.ID}
	}
	if page.Version != update.ExpectedVersion {
		return RemotePage{}, &VersionConflictError{ID: update.ID, ExpectedVersion: update.ExpectedVersion}
	}
	if p.titleTaken(page.SpaceKey, update.Title, update.ID) {
		return RemotePage{}, &TitleTakenError{Title: update.Title}
	}
	page.Title, page.Body = update.Title, update.Body
	if update.ParentID != "" {
		page.ParentID = update.ParentID
	}
	page.Version++

	return page.RemotePage, nil
}

// MarkPage implements DocumentationPlatform.
func (p *InMemoryPlatform) MarkPage(_ context.Context, id string, sourcePath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("MarkPage %s %s", id, sourcePath)
	page, ok := p.pages[id]
	if !ok {
		return &PageNotFoundError{ID: id}
	}
	page.Marked, page.SourcePath = true, sourcePath

	return nil
}

// ListMarkedDescendants implements DocumentationPlatform.
func (p *InMemoryPlatform) ListMarkedDescendants(_ context.Context, rootID string) ([]RemotePage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("ListMarkedDescendants %s", rootID)
	var found []RemotePage
	for _, id := range p.sortedIDs() {
		if page := p.pages[id]; page.Marked && p.below(id, rootID) {
			found = append(found, page.RemotePage)
		}
	}

	return found, nil
}

func (p *InMemoryPlatform) below(id string, rootID string) bool {
	for seen := 0; seen <= len(p.pages); seen++ {
		page, ok := p.pages[id]
		if !ok || page.ParentID == "" {
			return false
		}
		if page.ParentID == rootID {
			return true
		}
		id = page.ParentID
	}

	return false
}

// TrashPage implements DocumentationPlatform.
func (p *InMemoryPlatform) TrashPage(_ context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("TrashPage %s", id)
	delete(p.pages, id)

	return nil
}

// UploadFile implements DocumentationPlatform.
func (p *InMemoryPlatform) UploadFile(_ context.Context, pageID string, file File) (UploadedFile, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("UploadFile %s %s", pageID, file.Name)
	if _, ok := p.pages[pageID]; !ok {
		return UploadedFile{}, &PageNotFoundError{ID: pageID}
	}
	sum := sha256.Sum256(file.Content)
	hash := "sha256:" + hex.EncodeToString(sum[:])
	files := p.attachments[pageID]
	if files == nil {
		files = map[string]UploadedFile{}
		p.attachments[pageID] = files
	}
	if existing, ok := files[file.Name]; ok && existing.Hash == hash {
		existing.Skipped = true

		return existing, nil
	}
	uploaded := UploadedFile{ID: "att-" + pageID + "-" + file.Name, Name: file.Name, Hash: hash}
	files[file.Name] = uploaded
	if p.attachmentContent[pageID] == nil {
		p.attachmentContent[pageID] = map[string][]byte{}
	}
	p.attachmentContent[pageID][file.Name] = file.Content

	return uploaded, nil
}

// ListAttachments implements DocumentationPlatform.
func (p *InMemoryPlatform) ListAttachments(_ context.Context, pageID string) ([]RemoteAttachment, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("ListAttachments %s", pageID)
	if _, ok := p.pages[pageID]; !ok {
		return nil, &PageNotFoundError{ID: pageID}
	}
	out := make([]RemoteAttachment, 0, len(p.attachments[pageID]))
	for name, file := range p.attachments[pageID] {
		out = append(out, RemoteAttachment{Filename: name, Hash: file.Hash})
	}
	slices.SortFunc(out, func(a, b RemoteAttachment) int { return strings.Compare(a.Filename, b.Filename) })

	return out, nil
}

// DownloadAttachment implements DocumentationPlatform.
func (p *InMemoryPlatform) DownloadAttachment(_ context.Context, pageID string, filename string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log("DownloadAttachment %s %s", pageID, filename)
	content, ok := p.attachmentContent[pageID][filename]
	if !ok {
		return nil, &PageNotFoundError{ID: pageID + "/" + filename}
	}

	return content, nil
}
