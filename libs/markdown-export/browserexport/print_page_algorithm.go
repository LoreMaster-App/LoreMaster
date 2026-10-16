package browserexport

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PageSizes are the paper sizes a PDF can be printed on.
var PageSizes = []string{"A4", "A3", "A5", "Letter", "Legal"}

// pageStyle is the print rule: the paper, and the margin around the text.
func pageStyle(size string) string {
	return fmt.Sprintf("<style>@page{size:%s;margin:18mm}</style>", size)
}

// lightTheme makes the page light whatever the machine prefers: an export is the same on every computer,
// and a dark PNG or PDF is not what anyone sends. It comes after the page's own styles, so it wins.
const lightTheme = "<style>html:root{color-scheme:light;--fg:#1f2328;--bg:#fff;--muted:#59636e;--rule:#d1d9e0;--code:#f6f8fa;--link:#0969da}</style>"

// withHeadBefore puts markup at the end of the page's head.
func withHeadBefore(page, markup string) string {
	if index := strings.Index(page, "</head>"); index >= 0 {
		return page[:index] + markup + page[index:]
	}

	return markup + page
}

// heightProbe records the height of the page's content in a meta tag once it has loaded, so a screenshot
// can be taken of the whole page and not only of what fits the window. The window is no guide: it is never
// shorter than the one the page was opened in. The browser prints the page's markup when asked to dump it,
// and the tag is read back from there.
const heightProbe = `<script>
function record(){var m=document.createElement('meta');m.name='lore-master-height';m.content=String(Math.ceil(document.body.getBoundingClientRect().bottom));document.head.appendChild(m)}
if(document.readyState==='complete'){record()}else{window.addEventListener('load',record)}
</script>`

var recordedHeight = regexp.MustCompile(`name="lore-master-height"\s+content="(\d+)"|content="(\d+)"\s+name="lore-master-height"`)

// pageHeight reads the height the probe recorded from a dump of the page.
func pageHeight(dump string) (int, bool) {
	match := recordedHeight.FindStringSubmatch(dump)
	if match == nil {
		return 0, false
	}
	digits := match[1]
	if digits == "" {
		digits = match[2]
	}
	height, err := strconv.Atoi(digits)

	return height, err == nil && height > 0
}

// validPageSize reports the size to print on: the one asked for when it is known, A4 otherwise.
func validPageSize(size string) string {
	for _, known := range PageSizes {
		if strings.EqualFold(known, size) {
			return known
		}
	}

	return "A4"
}
