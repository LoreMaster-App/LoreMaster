/** The webview page of the storage form: markup, styles and script, under one nonce. */
export function renderStorageFormPage (nonce: string, title: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'nonce-${nonce}'; script-src 'nonce-${nonce}';" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<style nonce="${nonce}">
  :root { color-scheme: light dark; }
  body { font-family: var(--vscode-font-family); color: var(--vscode-foreground); background: var(--vscode-editor-background); padding: 16px 24px; max-width: 760px; }
  h1 { font-size: 1.35em; font-weight: 600; margin: 0 0 4px; }
  p.lead { margin: 0 0 16px; color: var(--vscode-descriptionForeground); }
  section { border: 1px solid var(--vscode-panel-border); border-radius: 6px; padding: 12px 16px; margin: 0 0 16px; }
  section h2 { font-size: 1em; margin: 0 0 8px; text-transform: uppercase; letter-spacing: .04em; color: var(--vscode-descriptionForeground); }
  label { display: block; margin: 10px 0 2px; font-weight: 600; }
  .hint { color: var(--vscode-descriptionForeground); font-size: .9em; margin: 2px 0 0; }
  input, select { width: 100%; box-sizing: border-box; padding: 5px 8px; color: var(--vscode-input-foreground); background: var(--vscode-input-background); border: 1px solid var(--vscode-input-border, transparent); border-radius: 2px; font: inherit; }
  input:focus, select:focus, button:focus { outline: 1px solid var(--vscode-focusBorder); }
  .row { display: flex; gap: 8px; align-items: stretch; }
  .row > :first-child { flex: 1; }
  button { padding: 5px 12px; border: 1px solid transparent; border-radius: 2px; font: inherit; cursor: pointer; color: var(--vscode-button-secondaryForeground); background: var(--vscode-button-secondaryBackground); }
  button.primary { color: var(--vscode-button-foreground); background: var(--vscode-button-background); }
  button:disabled { opacity: .5; cursor: default; }
  .error { color: var(--vscode-errorForeground); font-size: .9em; margin: 2px 0 0; }
  .banner { border: 1px solid var(--vscode-inputValidation-errorBorder); background: var(--vscode-inputValidation-errorBackground); padding: 8px 12px; margin: 0 0 16px; border-radius: 4px; }
  ul.results { list-style: none; margin: 6px 0 0; padding: 0; border: 1px solid var(--vscode-panel-border); border-radius: 4px; max-height: 180px; overflow: auto; }
  ul.results li { padding: 5px 8px; cursor: pointer; display: flex; justify-content: space-between; gap: 12px; }
  ul.results li:hover, ul.results li:focus { background: var(--vscode-list-hoverBackground); }
  ul.results li span.id { color: var(--vscode-descriptionForeground); }
  .row.tight { margin-top: 6px; }
  .actions { display: flex; gap: 8px; justify-content: flex-end; }
  [hidden] { display: none !important; }
</style>
<title>${title}</title>
</head>
<body>
<h1 id="heading">${title}</h1>
<p class="lead" id="lead"></p>
<div class="banner" id="banner" hidden></div>
<form id="form" novalidate>
  <section id="sec-platform">
    <h2>Where to sync</h2>
    <label for="platform">Storage</label>
    <select id="platform">
      <option value="">Choose a storage…</option>
      <option value="confluence">Confluence: Markdown to a Confluence space</option>
      <option value="github-pages">GitHub Pages: a static site on a branch</option>
      <option value="github-wiki">GitHub Wiki: the repository's wiki pages</option>
    </select>
    <div class="error" data-error="platform"></div>
  </section>

  <section id="sec-confluence" hidden>
    <h2>Confluence</h2>
    <label for="connection">Connection</label>
    <div class="row"><select id="connection"></select><button type="button" id="addConnection">Add connection…</button></div>
    <div class="error" data-error="connection"></div>

    <label for="space">Space</label>
    <div class="row"><input id="space" autocomplete="off" placeholder="Space key, e.g. ENG" /><button type="button" id="loadSpaces">Browse spaces</button></div>
    <select id="spaceList" hidden size="6"></select>
    <div class="error" data-error="space"></div>

    <label for="parentPageId">Parent page</label>
    <input id="parentPageId" autocomplete="off" placeholder="Page id, e.g. 98306" />
    <p class="hint" id="parentTitle">Every synced page goes under this page. Changing it moves the pages already synced.</p>
    <div class="row tight"><input id="pageQuery" autocomplete="off" placeholder="Find the parent page by title" /><button type="button" id="searchPages">Search</button></div>
    <ul class="results" id="pageResults" hidden></ul>
    <div class="error" data-error="parentPageId"></div>

    <label for="titlePrefix">Title prefix</label>
    <input id="titlePrefix" autocomplete="off" placeholder="Pages are titled “&lt;prefix&gt;: &lt;first heading&gt;”" />
    <p class="hint">Leave empty to ask on the first sync.</p>

    <label for="direction">Direction</label>
    <select id="direction"><option value="to-platform">to-platform: Markdown is the source</option><option value="two-way">two-way: pull edits back</option></select>
    <label for="mermaidMode">Mermaid diagrams</label>
    <select id="mermaidMode"><option value="image">image</option><option value="code">code</option></select>
    <label for="linkMode">Links between pages</label>
    <select id="linkMode"><option value="title">title</option><option value="id">id</option></select>
    <label for="titleCollision">When a page title already exists</label>
    <select id="titleCollision"><option value="fail">fail</option><option value="adopt">adopt</option></select>
  </section>

  <section id="sec-github" hidden>
    <h2 id="githubHeading">GitHub</h2>
    <label for="repo">Repository</label>
    <input id="repo" autocomplete="off" placeholder="owner/name or a clone URL" />
    <p class="hint">Leave empty for this workspace's own origin.</p>
    <div class="error" data-error="repo"></div>
    <label for="branch">Branch</label>
    <input id="branch" autocomplete="off" placeholder="gh-pages" />
    <div id="pathRow">
      <label for="path">Folder in the branch</label>
      <input id="path" autocomplete="off" placeholder="(the whole branch)" />
      <p class="hint">Publish into this folder and leave the rest of the branch alone.</p>
    </div>
  </section>

  <section id="sec-common" hidden>
    <h2>What to sync</h2>
    <label for="roots">Folders to sync</label>
    <input id="roots" autocomplete="off" placeholder="." />
    <p class="hint">Folders relative to the workspace, separated by commas. “.” is the whole workspace.</p>
    <label for="excludes">Excluded paths</label>
    <input id="excludes" autocomplete="off" placeholder="drafts/, CHANGELOG.md" />
    <p class="hint">Gitignore-style patterns this storage leaves out.</p>
    <label for="template">Page template</label>
    <input id="template" autocomplete="off" placeholder="default" />
    <p class="hint">Empty is the default layout. GitHub Pages: default, minimal, handbook or headless, or a folder holding template.html. Wiki: a Markdown file with {{content}}. Confluence: a page id or address, or a file, with {{content}} where the page goes. Write ./name for a folder named like a built-in.</p>
    <div class="error" data-error="template"></div>
    <label for="ignore">Ignored paths (all storages)</label>
    <input id="ignore" autocomplete="off" placeholder="internal/, CLAUDE.md" />
    <p class="hint">Patterns every storage of this workspace leaves out.</p>
  </section>

  <div class="actions">
    <button type="button" id="cancel">Cancel</button>
    <button type="submit" class="primary" id="save">Save</button>
  </div>
</form>

<script nonce="${nonce}">
(function () {
  const api = acquireVsCodeApi();
  const $ = id => document.getElementById(id);
  const FIELDS = ['platform','connection','space','parentPageId','titlePrefix','direction','mermaidMode','linkMode','titleCollision','repo','branch','path','template','roots','excludes','ignore'];
  let mode = 'create';

  function read () {
    const values = {};
    for (const name of FIELDS) { values[name] = $(name).value; }
    return values;
  }
  function write (values) {
    for (const name of FIELDS) {
      if (values[name] === undefined || (name === 'connection' && values[name] === '')) { continue; }
      $(name).value = values[name];
    }
  }
  function clearErrors () {
    document.querySelectorAll('[data-error]').forEach(node => { node.textContent = ''; });
    $('banner').hidden = true;
  }
  function showErrors (errors, message) {
    clearErrors();
    for (const [field, text] of Object.entries(errors || {})) {
      const node = document.querySelector('[data-error="' + field + '"]');
      if (node) { node.textContent = text; }
    }
    if (message) { $('banner').textContent = message; $('banner').hidden = false; }
  }
  function layout () {
    const platform = $('platform').value;
    $('sec-confluence').hidden = platform !== 'confluence';
    $('sec-github').hidden = platform !== 'github-pages' && platform !== 'github-wiki';
    $('sec-common').hidden = platform === '';
    $('pathRow').hidden = platform !== 'github-pages';
    $('githubHeading').textContent = platform === 'github-wiki' ? 'GitHub Wiki' : 'GitHub Pages';
    $('branch').placeholder = platform === 'github-wiki' ? "(the wiki's default branch)" : 'gh-pages';
  }
  function fillConnections (connections, selected) {
    const select = $('connection');
    select.textContent = '';
    if (connections.length === 0) {
      const none = document.createElement('option'); none.value = ''; none.textContent = 'No connection yet: add one'; select.appendChild(none);
    }
    for (const connection of connections) {
      const option = document.createElement('option'); option.value = connection.baseUrl; option.textContent = connection.label; select.appendChild(option);
    }
    if (selected) { select.value = selected; } else if (connections.length > 0) { select.value = connections[0].baseUrl; }
  }
  function fillList (list, items, label, choose) {
    list.textContent = '';
    list.hidden = items.length === 0;
    for (const item of items) {
      const entry = document.createElement('li'); entry.tabIndex = 0;
      const name = document.createElement('span'); name.textContent = label(item.title);
      const id = document.createElement('span'); id.className = 'id'; id.textContent = item.id;
      entry.append(name, id);
      const pick = () => choose(item);
      entry.addEventListener('click', pick);
      entry.addEventListener('keydown', event => { if (event.key === 'Enter') { pick(); } });
      list.appendChild(entry);
    }
  }

  $('platform').addEventListener('change', layout);
  $('addConnection').addEventListener('click', () => api.postMessage({ type: 'addConnection' }));
  $('loadSpaces').addEventListener('click', () => { $('loadSpaces').disabled = true; api.postMessage({ type: 'listSpaces', connection: $('connection').value }); });
  $('spaceList').addEventListener('change', () => { $('space').value = $('spaceList').value; });
  $('searchPages').addEventListener('click', () => {
    $('searchPages').disabled = true;
    api.postMessage({ type: 'searchPages', connection: $('connection').value, space: $('space').value, query: $('pageQuery').value });
  });
  $('pageQuery').addEventListener('keydown', event => { if (event.key === 'Enter') { event.preventDefault(); $('searchPages').click(); } });
  $('cancel').addEventListener('click', () => api.postMessage({ type: 'cancel' }));
  $('form').addEventListener('submit', event => {
    event.preventDefault();
    clearErrors();
    $('save').disabled = true;
    api.postMessage({ type: 'save', values: read() });
  });

  window.addEventListener('message', event => {
    const message = event.data;
    switch (message.type) {
      case 'init': {
        mode = message.init.mode;
        $('heading').textContent = mode === 'edit' ? 'Edit storage' : 'Add a storage';
        $('lead').textContent = mode === 'edit' ? 'Change anything and save: the changes apply at the next sync.' : 'Fill the form and save: nothing is synced until you run a sync.';
        fillConnections(message.init.connections, message.init.values.connection);
        write(message.init.values);
        if (mode === 'edit') { $('platform').disabled = true; }
        layout();
        break;
      }
      case 'connections':
        fillConnections(message.connections, message.selected);
        break;
      case 'spaces': {
        $('loadSpaces').disabled = false;
        const list = $('spaceList');
        list.textContent = '';
        for (const space of message.spaces) {
          const option = document.createElement('option'); option.value = space.key; option.textContent = space.key + ': ' + space.name; list.appendChild(option);
        }
        list.hidden = message.spaces.length === 0;
        break;
      }
      case 'pages':
        $('searchPages').disabled = false;
        fillList($('pageResults'), message.pages, title => title, page => {
          $('parentPageId').value = page.id;
          $('parentTitle').textContent = 'Parent: ' + page.title + ' (' + page.id + ')';
          if ($('titlePrefix').value === '') { $('titlePrefix').value = page.title; }
          $('pageResults').hidden = true;
        });
        break;
      case 'lookupError':
        $('loadSpaces').disabled = false;
        $('searchPages').disabled = false;
        $('banner').textContent = message.message;
        $('banner').hidden = false;
        break;
      case 'invalid':
        $('save').disabled = false;
        showErrors(message.errors, message.message);
        break;
      default:
        break;
    }
  });

  api.postMessage({ type: 'ready' });
})();
</script>
</body>
</html>`
}
