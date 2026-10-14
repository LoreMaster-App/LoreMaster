package storybookdocs

import "testing"

func TestParseStoryIndexListsStoriesOrderedByTitleAndID(t *testing.T) {
	data := []byte(`{"v":5,"entries":{
		"b--two":{"id":"b--two","type":"story","name":"Two","title":"Components / B","importPath":"./b.stories.tsx"},
		"a--primary":{"id":"a--primary","type":"story","name":"Primary","title":"Components/A","importPath":"./a.stories.tsx"},
		"a--docs":{"id":"a--docs","type":"docs","name":"Docs","title":"Components/A"},
		"a--base":{"id":"a--base","type":"story","name":"Base","title":"Components/A"}
	}}`)
	stories, err := ParseStoryIndex(data)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, story := range stories {
		got = append(got, story.Title+"|"+story.ID)
	}
	want := []string{"Components/A|a--base", "Components/A|a--primary", "Components/B|b--two"}
	if len(got) != len(want) {
		t.Fatalf("stories = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stories = %v, want %v", got, want)
		}
	}
}

func TestParseStoryIndexReadsTheOlderStoriesShape(t *testing.T) {
	data := []byte(`{"v":3,"stories":{"x--y":{"id":"x--y","kind":"Widgets/X","story":"Y","importPath":"./x.js"}}}`)
	stories, err := ParseStoryIndex(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(stories) != 1 || stories[0].Title != "Widgets/X" || stories[0].Name != "Y" {
		t.Fatalf("stories = %+v", stories)
	}
}

func TestParseStoryIndexRefusesAnIndexWithoutStories(t *testing.T) {
	if _, err := ParseStoryIndex([]byte(`{"v":5,"entries":{}}`)); err == nil {
		t.Fatal("want an error for an empty index")
	}
	if _, err := ParseStoryIndex([]byte(`not json`)); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}
