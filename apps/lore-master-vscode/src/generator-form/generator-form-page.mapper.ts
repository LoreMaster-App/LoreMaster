/** The webview page of the generator form: markup, styles and script, under one nonce. */
export function renderGeneratorFormPage (nonce: string, title: string): string {
  return String.raw`<!DOCTYPE html>
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
  button { padding: 5px 12px; border: 1px solid transparent; border-radius: 2px; font: inherit; cursor: pointer; color: var(--vscode-button-secondaryForeground); background: var(--vscode-button-secondaryBackground); }
  button.primary { color: var(--vscode-button-foreground); background: var(--vscode-button-background); }
  button:disabled { opacity: .5; cursor: default; }
  .error { color: var(--vscode-errorForeground); font-size: .9em; margin: 2px 0 0; }
  .banner { border: 1px solid var(--vscode-inputValidation-errorBorder); background: var(--vscode-inputValidation-errorBackground); padding: 8px 12px; margin: 0 0 16px; border-radius: 4px; }
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
  <section>
    <h2>What to write</h2>
    <label for="kind">Generator</label>
    <select id="kind"></select>
    <p class="hint" id="kindHint"></p>
    <div class="error" data-error="kind"></div>
  </section>

  <section id="sec-details" hidden>
    <h2>Where it goes and what it reads</h2>
    <label for="output">Output folder</label>
    <input id="output" autocomplete="off" placeholder="docs/generated" />
    <p class="hint" id="outputHint">A folder for the generated pages, relative to the workspace. Only this generator writes to it.</p>
    <div class="error" data-error="output"></div>

    <label for="reads" id="readsLabel">What it reads</label>
    <input id="reads" autocomplete="off" />
    <p class="hint" id="readsHint"></p>
    <div class="error" data-error="reads"></div>

    <label for="command">Command that produces it</label>
    <input id="command" autocomplete="off" placeholder="npm test -- --reporters=jest-junit" />
    <p class="hint" id="commandHint"></p>
    <div class="error" data-error="command"></div>

    <label for="title">Index page title</label>
    <input id="title" autocomplete="off" placeholder="(the default)" />
    <p class="hint">Leave empty for the default.</p>
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
  const FIELDS = ['kind','output','reads','command','title'];
  let mode = 'create';
  let kinds = [];
  let taken = [];
  let touched = { output: false, reads: false };

  function read () {
    const values = {};
    for (const name of FIELDS) { values[name] = $(name).value; }
    return values;
  }
  function current () { return kinds.find(kind => kind.kind === $('kind').value); }
  function freeFolder (base) {
    const used = new Set(taken.map(folder => folder.replace(/\/+$/, '')));
    let candidate = base;
    for (let n = 2; used.has(candidate); n += 1) { candidate = base + '-' + n; }
    return candidate;
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
  function layout (fresh) {
    const kind = current();
    $('sec-details').hidden = !kind;
    $('kindHint').textContent = kind ? kind.description : '';
    if (!kind) { return; }
    $('readsLabel').textContent = kind.readsSource ? 'Storybook to read' : 'What it reads';
    $('readsHint').textContent = kind.readsHint + (kind.readsRequired ? '' : '');
    $('reads').placeholder = kind.readsSource ? 'storybook-static or https://…' : (kind.readsRequired ? 'build/reports/' : '(the default)');
    $('commandHint').textContent = kind.command === 'required'
      ? 'A shell command run from the workspace. Required: this generator has nothing else to read. It runs when you generate, after you approve it.'
      : 'Optional. A shell command run before generating, e.g. npm run build-storybook. Leave empty to build it yourself.';
    if (fresh) {
      if (!touched.output) { $('output').value = freeFolder(kind.defaultOutput); }
      if (!touched.reads) { $('reads').value = kind.defaultReads; }
    }
  }

  $('kind').addEventListener('change', () => layout(true));
  $('output').addEventListener('input', () => { touched.output = true; });
  $('reads').addEventListener('input', () => { touched.reads = true; });
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
        kinds = message.init.kinds;
        taken = message.init.takenOutputs;
        $('heading').textContent = mode === 'edit' ? 'Edit generator' : 'Add a generator';
        $('lead').textContent = mode === 'edit'
          ? 'Change anything and save. The kind cannot change; remove the generator and add another for that.'
          : 'Choose what to write as pages. Nothing runs until you generate.';
        const select = $('kind');
        select.textContent = '';
        if (mode === 'create') {
          const none = document.createElement('option'); none.value = ''; none.textContent = 'Choose a generator…'; select.appendChild(none);
        }
        for (const kind of kinds) {
          const option = document.createElement('option'); option.value = kind.kind; option.textContent = kind.label; select.appendChild(option);
        }
        const values = message.init.values;
        for (const name of FIELDS) { if (values[name] !== undefined) { $(name).value = values[name]; } }
        if (mode === 'edit') { select.disabled = true; touched = { output: true, reads: true }; }
        layout(false);
        break;
      }
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
