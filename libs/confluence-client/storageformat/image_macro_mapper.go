package storageformat

import (
	"errors"
	"strconv"
)

// image writes <ac:image> around an attachment or a URL, with alt, title and width
// when given.
func (r *renderer) image(image Image) error {
	var resource string
	switch source := image.Source.(type) {
	case *AttachmentRef:
		if source == nil || source.Filename == "" {
			return errors.New("an attachment image needs a file name")
		}
		resource = `<ri:attachment ri:filename="` + escapeText(source.Filename) + `" />`
	case *URLRef:
		if source == nil || source.URL == "" {
			return errors.New("a URL image needs a URL")
		}
		resource = `<ri:url ri:value="` + escapeText(source.URL) + `" />`
	default:
		return errors.New("an image needs an attachment or a URL")
	}
	r.out.WriteString("<ac:image")
	if image.Alt != "" {
		r.out.WriteString(` ac:alt="` + escapeText(image.Alt) + `"`)
	}
	if image.Title != "" {
		r.out.WriteString(` ac:title="` + escapeText(image.Title) + `"`)
	}
	if image.Width > 0 {
		r.out.WriteString(` ac:width="` + strconv.Itoa(image.Width) + `"`)
	}
	r.out.WriteString(">" + resource + "</ac:image>")

	return nil
}
