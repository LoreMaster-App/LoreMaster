package configcatalog

import "lore-master/libs/documentation-sync/templatereference"

func confluencePlatform() Platform {
	return Platform{
		ID: "confluence", Label: "Confluence", Icon: "confluence", Delivery: DeliveryPages,
		Description:  "Atlassian Confluence: Cloud, Data Center or Server.",
		NameTemplate: "Confluence · {space|no space yet}",
		Connection: &Connection{
			Field:        "baseUrl",
			DetectMethod: "edition/detect",
			Kinds: []CredentialKind{
				{
					ID: "apitoken", Label: "Email and API token", Editions: []string{"cloud"},
					Help: "Atlassian Cloud: create the token at id.atlassian.com under Security > API tokens.",
					Fields: []CredentialField{
						{Key: "email", Label: "Email"},
						{Key: "token", Label: "API token", Secret: true},
					},
				},
				{
					ID: "pat", Label: "Personal access token", Editions: []string{"datacenter", "server"},
					Help:   "Data Center, and Server 7.9 or later: create it in your profile under Personal Access Tokens.",
					Fields: []CredentialField{{Key: "token", Label: "Token", Secret: true}},
				},
				{
					ID: "basic", Label: "User and password", Editions: []string{"server"},
					Help: "Server before 7.9, which has no personal access tokens.",
					Fields: []CredentialField{
						{Key: "user", Label: "User"},
						{Key: "password", Label: "Password", Secret: true},
					},
				},
			},
		},
		Fields: []Field{
			{
				Key: "baseUrl", Label: "Confluence address", Type: FieldText, Required: true,
				Placeholder: "https://example.atlassian.net/wiki",
				Help:        "The address of the site, https. Any page address of the site works.",
			},
			{
				Key: "space", Label: "Space", Type: FieldText, Required: true,
				Help:   "The space the pages go in.",
				Lookup: &Lookup{Kind: LookupSpace},
			},
			{
				Key: "parentPageId", Label: "Parent page", Type: FieldText, Required: true,
				Help:    "The page the tree hangs under.",
				Pattern: `^\d+$`, PatternHelp: "The parent page is a page id: digits only. Search for it by title to fill it in.",
				Lookup: &Lookup{
					Kind: LookupPage, Needs: []string{"space"},
					Fills: []Fill{{Field: "titlePrefix", From: "title"}},
				},
			},
			{
				Key: "titlePrefix", Label: "Title prefix", Type: FieldText, Required: true,
				Help: "Pages are titled \"<prefix>: <first heading>\". Starts as the parent page's title.",
			},
			{
				Key: "direction", Label: "Direction", Type: FieldChoice, Default: "to-platform", Advanced: true,
				Help: "Which way changes flow.",
				Choices: []Choice{
					{Value: "to-platform", Label: "Files to Confluence", Help: "Files are the source; the platform is overwritten."},
					{Value: "two-way", Label: "Both ways", Help: "Edits made on the platform are pulled back into the files."},
				},
			},
			{
				Key: "mermaidMode", Label: "Mermaid diagrams", Type: FieldChoice, Default: "image", Advanced: true,
				Choices: []Choice{
					{Value: "image", Label: "Image", Help: "A code macro plus an SVG attachment."},
					{Value: "code", Label: "Code only", Help: "A code macro only."},
				},
			},
			{
				Key: "titleCollision", Label: "Title already exists", Type: FieldChoice, Default: "fail", Advanced: true,
				Help: "What to do when an unsynced file's title already exists on the platform.",
				Choices: []Choice{
					{Value: "fail", Label: "Stop"},
					{Value: "adopt", Label: "Adopt the existing page"},
				},
			},
			{
				Key: "linkMode", Label: "Links between pages", Type: FieldChoice, Default: "title", Advanced: true,
				Choices: []Choice{
					{Value: "title", Label: "By page title"},
					{Value: "id", Label: "By page id"},
				},
			},
			{
				Key: "template", Label: "Template", Type: FieldText, Advanced: true,
				Help:        "How the pages are laid out: a page id or address, or a file, with {{content}} where the page goes. Empty is the default layout.",
				Suggestions: templatereference.BuiltinNames("confluence"),
			},
		},
	}
}
