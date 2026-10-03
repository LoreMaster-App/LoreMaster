package remotesync

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// fakeConfluence is a stateful stand-in for a Confluence Data Center site, speaking just
// enough of the v1 REST API for the sync to run against it: content create/read/update,
// CQL search (title lookup and the marked-descendants listing), the global label used as
// the sync's marker, and attachment upload. It keeps pages and attachments in memory and
// records every mutating request, so a test can assert that a re-run writes nothing.
//
// It is faithful to the one dialect the sync uses on Data Center; it is not a general
// Confluence emulator. The base path is "/confluence", matching the adapter's test setup.
type fakeConfluence struct {
	mu          sync.Mutex
	base        string
	pages       map[string]*fakePage
	attachments map[string]map[string]*fakeAttachment
	mutations   []string
	nextID      int
	nextAtt     int
}

type fakePage struct {
	id       string
	title    string
	spaceKey string
	parentID string
	version  int
	marked   bool
}

type fakeAttachment struct {
	id      string
	comment string
}

func newFakeConfluence() *fakeConfluence {
	return &fakeConfluence{pages: map[string]*fakePage{}, attachments: map[string]map[string]*fakeAttachment{}}
}

// seedPage puts a page on the site as if a person had made it (unmarked), and returns its
// id. Used for the parent the sync nests under, and for adopt scenarios.
func (f *fakeConfluence) seedPage(spaceKey string, parentID string, title string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.store(spaceKey, parentID, title).id
}

// mutationCount is the number of mutating requests served so far.
func (f *fakeConfluence) mutationCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.mutations)
}

func (f *fakeConfluence) page(id string) *fakePage {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.pages[id]
}

func (f *fakeConfluence) attachmentNames(pageID string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var names []string
	for name := range f.attachments[pageID] {
		names = append(names, name)
	}
	slices.Sort(names)

	return names
}

func (f *fakeConfluence) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/confluence/rest/api/")
	if rest == r.URL.Path {
		http.NotFound(w, r)

		return
	}
	seg := strings.Split(strings.Trim(rest, "/"), "/")
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case len(seg) == 1 && seg[0] == "content" && r.Method == http.MethodPost:
		f.create(w, r)
	case len(seg) == 2 && seg[0] == "content" && seg[1] == "search" && r.Method == http.MethodGet:
		f.search(w, r)
	case len(seg) == 2 && seg[0] == "content" && r.Method == http.MethodGet:
		f.get(w, seg[1])
	case len(seg) == 2 && seg[0] == "content" && r.Method == http.MethodPut:
		f.update(w, r, seg[1])
	case len(seg) == 3 && seg[0] == "content" && seg[2] == "label" && r.Method == http.MethodPost:
		f.label(w, seg[1])
	case len(seg) == 4 && seg[0] == "content" && seg[2] == "child" && seg[3] == "page" && r.Method == http.MethodGet:
		f.children(w, seg[1])
	case len(seg) == 4 && seg[0] == "content" && seg[2] == "child" && seg[3] == "attachment" && r.Method == http.MethodGet:
		f.listAttachments(w, r, seg[1])
	case len(seg) == 4 && seg[0] == "content" && seg[2] == "child" && seg[3] == "attachment" && r.Method == http.MethodPost:
		f.createAttachment(w, r, seg[1])
	case len(seg) == 6 && seg[0] == "content" && seg[3] == "attachment" && seg[5] == "data" && r.Method == http.MethodPost:
		f.updateAttachmentData(w, r, seg[1], seg[4])
	default:
		f.writeError(w, http.StatusNotFound, "unexpected "+r.Method+" "+r.URL.Path)
	}
}

func (f *fakeConfluence) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
		Space struct {
			Key string `json:"key"`
		} `json:"space"`
		Ancestors []struct {
			ID string `json:"id"`
		} `json:"ancestors"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	parent := ""
	if n := len(body.Ancestors); n > 0 {
		parent = body.Ancestors[n-1].ID
	}
	if f.titleTaken(body.Space.Key, body.Title, "") {
		f.writeError(w, http.StatusBadRequest, "A page with this title already exists: "+body.Title)

		return
	}
	page := f.store(body.Space.Key, parent, body.Title)
	f.mutations = append(f.mutations, "create "+page.title)
	f.writeJSON(w, http.StatusOK, f.pageJSON(page))
}

func (f *fakeConfluence) update(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Title     string `json:"title"`
		Ancestors []struct {
			ID string `json:"id"`
		} `json:"ancestors"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	page, ok := f.pages[id]
	if !ok {
		f.writeError(w, http.StatusNotFound, "No content found")

		return
	}
	if body.Version.Number != page.version+1 {
		f.writeError(w, http.StatusConflict, "Version must be incremented on update")

		return
	}
	if f.titleTaken(page.spaceKey, body.Title, id) {
		f.writeError(w, http.StatusBadRequest, "A page with this title already exists: "+body.Title)

		return
	}
	page.title = body.Title
	if n := len(body.Ancestors); n > 0 {
		page.parentID = body.Ancestors[n-1].ID
	}
	page.version = body.Version.Number
	f.mutations = append(f.mutations, "update "+page.title)
	f.writeJSON(w, http.StatusOK, f.pageJSON(page))
}

func (f *fakeConfluence) get(w http.ResponseWriter, id string) {
	page, ok := f.pages[id]
	if !ok {
		f.writeError(w, http.StatusNotFound, "No content found")

		return
	}
	f.writeJSON(w, http.StatusOK, f.pageJSON(page))
}

func (f *fakeConfluence) search(w http.ResponseWriter, r *http.Request) {
	cql := r.URL.Query().Get("cql")
	var results []map[string]any
	if strings.Contains(cql, `label = "lore-master"`) {
		root := cqlValue(cql, "ancestor = ")
		for _, id := range f.sortedIDs() {
			if page := f.pages[id]; page.marked && f.below(id, root) {
				results = append(results, f.pageJSON(page))
			}
		}
	} else {
		key, title := cqlValue(cql, "space = "), cqlValue(cql, "title = ")
		for _, id := range f.sortedIDs() {
			if page := f.pages[id]; page.spaceKey == key && strings.EqualFold(page.title, title) {
				results = append(results, f.pageJSON(page))
			}
		}
	}
	f.writeList(w, results)
}

func (f *fakeConfluence) children(w http.ResponseWriter, parentID string) {
	var results []map[string]any
	for _, id := range f.sortedIDs() {
		if f.pages[id].parentID == parentID {
			results = append(results, f.pageJSON(f.pages[id]))
		}
	}
	f.writeList(w, results)
}

func (f *fakeConfluence) label(w http.ResponseWriter, id string) {
	page, ok := f.pages[id]
	if !ok {
		f.writeError(w, http.StatusNotFound, "No content found")

		return
	}
	page.marked = true
	f.mutations = append(f.mutations, "mark "+page.title)
	f.writeJSON(w, http.StatusOK, map[string]any{})
}

func (f *fakeConfluence) listAttachments(w http.ResponseWriter, r *http.Request, pageID string) {
	filename := r.URL.Query().Get("filename")
	var results []map[string]any
	if attachment, ok := f.attachments[pageID][filename]; ok {
		results = append(results, map[string]any{
			"id": attachment.id, "title": filename,
			"metadata": map[string]any{"comment": attachment.comment},
			"version":  map[string]any{"number": 1},
		})
	}
	f.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (f *fakeConfluence) createAttachment(w http.ResponseWriter, r *http.Request, pageID string) {
	name, comment, ok := f.readUpload(w, r, pageID)
	if !ok {
		return
	}
	f.nextAtt++
	attachment := &fakeAttachment{id: "att" + strconv.Itoa(f.nextAtt), comment: comment}
	if f.attachments[pageID] == nil {
		f.attachments[pageID] = map[string]*fakeAttachment{}
	}
	f.attachments[pageID][name] = attachment
	f.mutations = append(f.mutations, "attach "+name)
	f.writeJSON(w, http.StatusOK, map[string]any{"results": []map[string]any{{"id": attachment.id, "title": name}}})
}

func (f *fakeConfluence) updateAttachmentData(w http.ResponseWriter, r *http.Request, pageID string, attID string) {
	comment := r.FormValue("comment")
	for name, attachment := range f.attachments[pageID] {
		if attachment.id == attID {
			attachment.comment = comment
			f.mutations = append(f.mutations, "reattach "+name)
			f.writeJSON(w, http.StatusOK, map[string]any{"id": attachment.id, "title": name})

			return
		}
	}
	f.writeError(w, http.StatusNotFound, "No content found")
}

func (f *fakeConfluence) readUpload(w http.ResponseWriter, r *http.Request, pageID string) (string, string, bool) {
	if _, ok := f.pages[pageID]; !ok {
		f.writeError(w, http.StatusNotFound, "No content found")

		return "", "", false
	}
	_, header, err := r.FormFile("file")
	if err != nil {
		f.writeError(w, http.StatusBadRequest, "no file part")

		return "", "", false
	}

	return header.Filename, r.FormValue("comment"), true
}

func (f *fakeConfluence) store(spaceKey string, parentID string, title string) *fakePage {
	f.nextID++
	id := "p" + strconv.Itoa(f.nextID)
	page := &fakePage{id: id, title: title, spaceKey: spaceKey, parentID: parentID, version: 1}
	f.pages[id] = page

	return page
}

func (f *fakeConfluence) titleTaken(spaceKey string, title string, except string) bool {
	for id, page := range f.pages {
		if id != except && page.spaceKey == spaceKey && strings.EqualFold(page.title, title) {
			return true
		}
	}

	return false
}

// below reports whether root is an ancestor of id at any depth.
func (f *fakeConfluence) below(id string, root string) bool {
	for seen := 0; seen <= len(f.pages); seen++ {
		page, ok := f.pages[id]
		if !ok || page.parentID == "" {
			return false
		}
		if page.parentID == root {
			return true
		}
		id = page.parentID
	}

	return false
}

func (f *fakeConfluence) sortedIDs() []string {
	ids := make([]string, 0, len(f.pages))
	for id := range f.pages {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int {
		ai, _ := strconv.Atoi(a[1:])
		bi, _ := strconv.Atoi(b[1:])

		return ai - bi
	})

	return ids
}

func (f *fakeConfluence) pageJSON(page *fakePage) map[string]any {
	out := map[string]any{
		"id": page.id, "title": page.title,
		"space":   map[string]any{"id": 1, "key": page.spaceKey},
		"version": map[string]any{"number": page.version},
		"_links":  map[string]any{"webui": "/display/" + page.spaceKey + "/" + page.id, "base": f.base},
	}
	if page.parentID != "" {
		out["ancestors"] = []map[string]any{{"id": page.parentID}}
	}

	return out
}

func (f *fakeConfluence) writeList(w http.ResponseWriter, results []map[string]any) {
	f.writeJSON(w, http.StatusOK, map[string]any{"results": results, "size": len(results), "limit": 250, "_links": map[string]any{}})
}

func (f *fakeConfluence) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (f *fakeConfluence) writeError(w http.ResponseWriter, status int, message string) {
	f.writeJSON(w, status, map[string]any{"message": message})
}

// cqlValue reads the quoted value that follows prefix in a CQL string, undoing the
// backslash escaping cqlString applies.
func cqlValue(cql string, prefix string) string {
	index := strings.Index(cql, prefix)
	if index < 0 {
		return ""
	}
	rest := cql[index+len(prefix):]
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	var value strings.Builder
	for i := 0; i < len(rest); i++ {
		if rest[i] == '\\' && i+1 < len(rest) {
			i++
			value.WriteByte(rest[i])

			continue
		}
		if rest[i] == '"' {
			break
		}
		value.WriteByte(rest[i])
	}

	return value.String()
}
