package storageformat

import "testing"

func TestChildrenRendersTheChildrenDisplayMacro(t *testing.T) {
	cases := map[string]Children{
		`<ac:structured-macro ac:name="children" ac:schema-version="2"></ac:structured-macro>`:                                                                                          {},
		`<ac:structured-macro ac:name="children" ac:schema-version="2"><ac:parameter ac:name="all">true</ac:parameter></ac:structured-macro>`:                                            {All: true},
		`<ac:structured-macro ac:name="children" ac:schema-version="2"><ac:parameter ac:name="depth">2</ac:parameter></ac:structured-macro>`:                                            {Depth: 2},
		`<ac:structured-macro ac:name="children" ac:schema-version="2"><ac:parameter ac:name="all">true</ac:parameter><ac:parameter ac:name="depth">3</ac:parameter></ac:structured-macro>`: {All: true, Depth: 3},
	}
	for want, block := range cases {
		got, err := Render(Document{Blocks: []Block{block}}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("render %+v:\n got  %s\n want %s", block, got, want)
		}
		doc, flags, err := Parse(want)
		if err != nil || len(flags) != 0 {
			t.Fatalf("parse %s: %v %v", want, err, flags)
		}
		if len(doc.Blocks) != 1 || doc.Blocks[0] != block {
			t.Errorf("parse %s = %+v, want %+v", want, doc.Blocks, block)
		}
	}
}

func TestAChildrenMacroWithOtherParametersStillParses(t *testing.T) {
	doc, flags, err := Parse(`<ac:structured-macro ac:name="children" ac:schema-version="2" ac:macro-id="x"><ac:parameter ac:name="sort">title</ac:parameter><ac:parameter ac:name="all">true</ac:parameter></ac:structured-macro>`)
	if err != nil || len(flags) != 0 {
		t.Fatalf("%v %v", err, flags)
	}
	if len(doc.Blocks) != 1 || doc.Blocks[0] != (Children{All: true}) {
		t.Fatalf("blocks = %+v", doc.Blocks)
	}
}
