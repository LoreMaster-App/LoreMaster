// Renders the pictures of the README and the Marketplace listing from assets/listing/listing.html
// with headless Chrome, Chromium or Edge. The Marketplace refuses SVG in a README, hence PNG.
//
//   node tools/listing-pictures.cjs            # all of them
//   LORE_MASTER_BROWSER=/path/to/chrome node tools/listing-pictures.cjs

const { execFileSync } = require('node:child_process')
const fs = require('node:fs')
const path = require('node:path')
const { pathToFileURL } = require('node:url')

const folder = path.join(__dirname, '..', 'assets', 'listing')
const page = pathToFileURL(path.join(folder, 'listing.html')).href
const pictures = [
  { name: 'how', height: 700 },
  { name: 'editor', height: 596 },
  { name: 'generators', height: 720 },
  { name: 'ci', height: 768 },
]

const candidates = [
  process.env.LORE_MASTER_BROWSER,
  'C:/Program Files/Google/Chrome/Application/chrome.exe',
  'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
  '/usr/bin/google-chrome',
  '/usr/bin/chromium',
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
].filter(Boolean)
const browser = candidates.find(candidate => fs.existsSync(candidate))
if (!browser) {
  throw new Error('No Chrome, Chromium or Edge found; set LORE_MASTER_BROWSER.')
}

for (const { name, height } of pictures) {
  const target = path.join(folder, `${name}.png`)
  execFileSync(browser, [
    '--headless=new',
    '--disable-gpu',
    '--hide-scrollbars',
    '--force-device-scale-factor=1',
    `--window-size=1600,${height}`,
    `--screenshot=${target}`,
    `${page}#${name}`,
  ], { stdio: 'ignore' })
  console.log(`wrote ${path.relative(process.cwd(), target)}`)
}
