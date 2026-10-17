/**
 * The webview page of the storage form: styles and one script, under one nonce. The page knows no
 * platform: the script builds every section and field from the description the extension sends
 * with `init` (the engine's `config/describe`), so a platform the engine adds appears here without a change.
 */
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
  details { margin: 12px 0 0; }
  summary { cursor: pointer; color: var(--vscode-descriptionForeground); }
  label { display: block; margin: 10px 0 2px; font-weight: 600; }
  .hint { color: var(--vscode-descriptionForeground); font-size: .9em; margin: 2px 0 0; }
  input, select { width: 100%; box-sizing: border-box; padding: 5px 8px; color: var(--vscode-input-foreground); background: var(--vscode-input-background); border: 1px solid var(--vscode-input-border, transparent); border-radius: 2px; font: inherit; }
  input[type=checkbox] { width: auto; }
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
  <section id="sec-platform"><h2>Where to sync</h2></section>
  <section id="sec-fields" hidden><h2 id="platformHeading"></h2></section>
  <section id="sec-common" hidden><h2>What to sync</h2></section>
  <div class="actions">
    <button type="button" id="cancel">Cancel</button>
    <button type="submit" class="primary" id="save">Save</button>
  </div>
</form>

<script nonce="${nonce}">
(function () {
  const api = acquireVsCodeApi();
  const $ = id => document.getElementById(id);
  let description = null;
  let connections = [];
  let mode = 'create';
  // The values of every field, kept across a change of platform; the inputs are read into it before a redraw.
  let values = {};
  let current = [];

  function el (tag, props, children) {
    const node = document.createElement(tag);
    Object.assign(node, props || {});
    for (const child of children || []) { node.append(child); }
    return node;
  }
  function hint (text) { return el('p', { className: 'hint', textContent: text }); }

  function platformOf (id) { return description.platforms.find(platform => platform.id === id); }

  function remember () {
    for (const entry of current) { values[entry.key] = entry.read(); }
  }

  // A field is drawn as its type says; a lookup kind this page does not know is a plain input.
  function drawField (field, platform) {
    const wrap = el('div', {});
    const id = 'f-' + field.key;
    wrap.append(el('label', { htmlFor: id, textContent: field.label + (field.required ? ' *' : '') }));
    const isConnection = platform && platform.connection && platform.connection.field === field.key;
    let control;
    if (isConnection) {
      control = el('select', { id });
      const add = el('button', { type: 'button', textContent: 'Add connection…' });
      add.addEventListener('click', () => api.postMessage({ type: 'addConnection' }));
      wrap.append(el('div', { className: 'row' }, [control, add]));
      fillConnections(control, values[field.key]);
    } else if (field.type === 'choice') {
      control = el('select', { id });
      for (const choice of field.choices || []) {
        control.append(el('option', { value: choice.value, textContent: choice.label, title: choice.help || '' }));
      }
      wrap.append(control);
    } else if (field.type === 'bool') {
      control = el('input', { id, type: 'checkbox', checked: values[field.key] === 'true' });
      wrap.append(control);
    } else {
      control = el('input', { id, autocomplete: 'off', placeholder: field.placeholder || field.default || '' });
      if (field.type === 'number') { control.type = 'number'; }
      if (field.suggestions && field.suggestions.length > 0) {
        const list = el('datalist', { id: id + '-list' });
        for (const suggestion of field.suggestions) { list.append(el('option', { value: suggestion })); }
        control.setAttribute('list', id + '-list');
        wrap.append(control, list);
      } else if (!field.lookup || !drawLookup(wrap, field, control)) {
        wrap.append(control);
      }
    }
    if (control.tagName === 'INPUT' && control.type === 'checkbox') {
      control.checked = values[field.key] === 'true';
    } else if (!isConnection) {
      control.value = values[field.key] || '';
    }
    if (field.help) { wrap.append(hint(field.help)); }
    wrap.append(el('div', { className: 'error' }));
    wrap.lastChild.setAttribute('data-error', field.key);
    current.push({
      key: field.key,
      read: () => control.type === 'checkbox' ? String(control.checked) : control.value,
      set: value => { control.value = value; },
    });
    return wrap;
  }

  // A lookup the page knows how to offer: a space is browsed, a page is searched in a space chosen first.
  function drawLookup (wrap, field, control) {
    if (field.lookup.kind === 'space') {
      const list = el('select', { size: 6, hidden: true });
      const browse = el('button', { type: 'button', textContent: 'Browse spaces' });
      browse.addEventListener('click', () => { browse.disabled = true; api.postMessage({ type: 'listSpaces', connection: connectionValue() }); });
      list.addEventListener('change', () => { control.value = list.value; });
      wrap.append(el('div', { className: 'row' }, [control, browse]), list);
      lookups.spaces = { list, browse };
      return true;
    }
    if (field.lookup.kind === 'page') {
      const query = el('input', { autocomplete: 'off', placeholder: 'Find it by title' });
      const search = el('button', { type: 'button', textContent: 'Search' });
      const results = el('ul', { className: 'results', hidden: true });
      const needs = (field.lookup.needs || [])[0];
      search.addEventListener('click', () => {
        search.disabled = true;
        remember();
        api.postMessage({ type: 'searchPages', connection: connectionValue(), space: needs ? (values[needs] || '') : '', query: query.value });
      });
      query.addEventListener('keydown', event => { if (event.key === 'Enter') { event.preventDefault(); search.click(); } });
      wrap.append(control, el('div', { className: 'row tight' }, [query, search]), results);
      lookups.pages = { results, search, control, fills: field.lookup.fills || [] };
      return true;
    }
    return false;
  }
  const lookups = {};

  function connectionValue () {
    const platform = platformOf($('platform').value);
    const key = platform && platform.connection ? platform.connection.field : '';
    const entry = current.find(each => each.key === key);
    return entry ? entry.read() : '';
  }

  function fillConnections (select, selected) {
    select.textContent = '';
    if (connections.length === 0) { select.append(el('option', { value: '', textContent: 'No connection yet: add one' })); }
    for (const connection of connections) { select.append(el('option', { value: connection.baseUrl, textContent: connection.label })); }
    if (selected) { select.value = selected; } else if (connections.length > 0) { select.value = connections[0].baseUrl; }
  }

  function drawFields (container, fields, platform) {
    const basic = fields.filter(field => !field.advanced);
    const advanced = fields.filter(field => field.advanced);
    for (const field of basic) { container.append(drawField(field, platform)); }
    if (advanced.length > 0) {
      const more = el('details', {}, [el('summary', { textContent: 'More options' })]);
      for (const field of advanced) { more.append(drawField(field, platform)); }
      container.append(more);
    }
  }

  function redraw () {
    const platform = platformOf($('platform').value);
    const secField = $('sec-fields');
    const secCommon = $('sec-common');
    current = current.filter(entry => entry.fixed);
    for (const key of Object.keys(lookups)) { delete lookups[key]; }
    secField.querySelectorAll('div, details').forEach(node => node.remove());
    secCommon.querySelectorAll('div, details').forEach(node => node.remove());
    secField.hidden = !platform;
    secCommon.hidden = !platform;
    if (!platform) { return; }
    $('platformHeading').textContent = platform.label;
    const own = el('div', {});
    drawFields(own, platform.fields, platform);
    secField.append(own);
    const common = el('div', {});
    const shared = description.contentFields.concat(description.ignoreField ? [Object.assign({}, description.ignoreField, { help: 'Patterns every storage of this workspace leaves out.' })] : []);
    drawFields(common, shared.concat(description.outputFields.filter(field => field.key !== 'name')), null);
    secCommon.append(common);
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

  // The parts that do not depend on the platform: the name and the platform choice.
  function drawStatic () {
    const section = $('sec-platform');
    const nameField = description.outputFields.find(field => field.key === 'name');
    if (nameField) { section.append(drawField(nameField, null)); current[current.length - 1].fixed = true; }
    const select = el('select', { id: 'platform' }, [el('option', { value: '', textContent: 'Choose a storage…' })]);
    for (const platform of description.platforms) {
      select.append(el('option', { value: platform.id, textContent: platform.label + ': ' + platform.description }));
    }
    select.value = values.platform || '';
    select.addEventListener('change', () => { remember(); redraw(); });
    section.append(el('label', { htmlFor: 'platform', textContent: 'Storage' }), select, el('div', { className: 'error' }));
    section.lastChild.setAttribute('data-error', 'platform');
    current.push({ key: 'platform', fixed: true, read: () => select.value, set: value => { select.value = value; } });
  }

  $('cancel').addEventListener('click', () => api.postMessage({ type: 'cancel' }));
  $('form').addEventListener('submit', event => {
    event.preventDefault();
    clearErrors();
    remember();
    $('save').disabled = true;
    api.postMessage({ type: 'save', values });
  });

  window.addEventListener('message', event => {
    const message = event.data;
    switch (message.type) {
      case 'init': {
        description = message.init.description;
        connections = message.init.connections;
        mode = message.init.mode;
        values = Object.assign({}, message.init.values);
        $('heading').textContent = mode === 'edit' ? 'Edit storage' : 'Add a storage';
        $('lead').textContent = mode === 'edit' ? 'Change anything and save: the changes apply at the next sync.' : 'Fill the form and save: nothing is synced until you run a sync.';
        drawStatic();
        if (mode === 'edit') { $('platform').disabled = true; }
        redraw();
        break;
      }
      case 'connections': {
        connections = message.connections;
        const platform = platformOf($('platform').value);
        const entry = platform && platform.connection ? current.find(each => each.key === platform.connection.field) : null;
        const select = entry ? $('f-' + entry.key) : null;
        if (select) { fillConnections(select, message.selected); }
        break;
      }
      case 'spaces': {
        const spaces = lookups.spaces;
        if (!spaces) { break; }
        spaces.browse.disabled = false;
        spaces.list.textContent = '';
        for (const space of message.spaces) { spaces.list.append(el('option', { value: space.key, textContent: space.key + ': ' + space.name })); }
        spaces.list.hidden = message.spaces.length === 0;
        break;
      }
      case 'pages': {
        const pages = lookups.pages;
        if (!pages) { break; }
        pages.search.disabled = false;
        pages.results.textContent = '';
        pages.results.hidden = message.pages.length === 0;
        for (const page of message.pages) {
          const entry = el('li', { tabIndex: 0 }, [el('span', { textContent: page.title }), el('span', { className: 'id', textContent: page.id })]);
          const pick = () => {
            pages.control.value = page.id;
            for (const fill of pages.fills) {
              const target = current.find(each => each.key === fill.field);
              if (target && target.read() === '') { target.set(fill.from === 'id' ? page.id : page.title); }
            }
            pages.results.hidden = true;
          };
          entry.addEventListener('click', pick);
          entry.addEventListener('keydown', key => { if (key.key === 'Enter') { pick(); } });
          pages.results.append(entry);
        }
        break;
      }
      case 'lookupError': {
        if (lookups.spaces) { lookups.spaces.browse.disabled = false; }
        if (lookups.pages) { lookups.pages.search.disabled = false; }
        $('banner').textContent = message.message;
        $('banner').hidden = false;
        break;
      }
      case 'invalid': {
        $('save').disabled = false;
        showErrors(message.errors, message.message);
        break;
      }
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
