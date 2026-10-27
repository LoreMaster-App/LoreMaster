package syncexecution

import "testing"

func TestPixelSizedSVGGivesMermaidOutputItsViewBoxSize(t *testing.T) {
	in := `<svg id="d1" width="100%" xmlns="http://www.w3.org/2000/svg" viewBox="4 4 1450 1176.24" style="max-width: 1450px;"><g/></svg>`
	want := `<svg id="d1" xmlns="http://www.w3.org/2000/svg" viewBox="4 4 1450 1176.24" style="max-width: 1450px;" width="1450" height="1176"><g/></svg>`
	if got := string(pixelSizedSVG([]byte(in))); got != want {
		t.Fatalf("got %s\nwant %s", got, want)
	}
}

func TestPixelSizedSVGLeavesSVGWithoutViewBoxAlone(t *testing.T) {
	in := `<svg width="10"><g/></svg>`
	if got := string(pixelSizedSVG([]byte(in))); got != in {
		t.Fatalf("got %s", got)
	}
}
