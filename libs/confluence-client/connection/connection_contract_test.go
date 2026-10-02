package connection

import "testing"

func TestParseVersion(t *testing.T) {
	cases := map[string]Version{
		"7.4":         {7, 4, 0},
		"8.5.3":       {8, 5, 3},
		"7.19.16-rc1": {7, 19, 16},
		"9.2.0.12345": {9, 2, 0},
	}
	for value, want := range cases {
		if got, err := ParseVersion(value); err != nil || got != want {
			t.Errorf("ParseVersion(%q) = %v, %v; want %v", value, got, err, want)
		}
	}
	for _, bad := range []string{"", "8", "eight.one", "8.x"} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("ParseVersion(%q) should fail", bad)
		}
	}
}

func TestVersionAtLeast(t *testing.T) {
	v := Version{7, 9, 0}
	if !v.AtLeast(7, 9) || v.AtLeast(7, 10) || !v.AtLeast(6, 99) || (Version{7, 4, 9}).AtLeast(7, 9) || !(Version{8, 0, 0}).AtLeast(7, 9) {
		t.Fatal("AtLeast compares major then minor")
	}
	if (Version{}).Known() || !v.Known() {
		t.Fatal("Known is false only for the zero version")
	}
}

func TestParseEdition(t *testing.T) {
	for _, value := range []string{"cloud", "datacenter", "server"} {
		if got, err := ParseEdition(value); err != nil || string(got) != value {
			t.Errorf("ParseEdition(%q) = %q, %v", value, got, err)
		}
	}
	if _, err := ParseEdition("Cloud"); err == nil || err.Error() != `unknown Confluence edition "Cloud"; expected cloud, datacenter or server` {
		t.Errorf("error %v", err)
	}
}
