package coveragereporting

import (
	"strings"
	"testing"
)

const coberturaSample = `<?xml version="1.0"?>
<!DOCTYPE coverage SYSTEM "http://cobertura.sourceforge.net/xml/coverage-04.dtd">
<coverage line-rate="0.5" branch-rate="0.5" version="1.9">
  <sources><source>C:\work\repo</source></sources>
  <packages>
    <package name="app">
      <classes>
        <class name="app.Cart" filename="src/cart.py">
          <methods><method name="add"><lines><line number="1" hits="1"/></lines></method></methods>
          <lines>
            <line number="1" hits="1" branch="false"/>
            <line number="2" hits="0" branch="true" condition-coverage="50% (1/2)"/>
            <line number="3" hits="4"/>
          </lines>
        </class>
        <class name="app.Cart.Item" filename="src/cart.py">
          <lines><line number="3" hits="2"/></lines>
        </class>
        <class name="app.Totals" filename="C:\work\repo\src\totals.py">
          <lines><line number="1" hits="0"/></lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>`

func TestParseCoberturaMergesClassesAndMakesPathsWorkspaceRelative(t *testing.T) {
	files, err := ParseCoverage([]byte(coberturaSample), `C:\work\repo`)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != "src/cart.py" || files[1].Path != "src/totals.py" {
		t.Fatalf("files = %+v", files)
	}
	covered, total := files[0].Lines()
	if covered != 2 || total != 3 {
		t.Fatalf("cart lines %d/%d, want 2/3 (the method's repeated line is not counted twice)", covered, total)
	}
	if files[0].LineHits[3] != 4 {
		t.Fatalf("line 3 hits = %d, want the higher count 4", files[0].LineHits[3])
	}
	branchCovered, branchTotal := files[0].BranchTotals()
	if branchCovered != 1 || branchTotal != 2 {
		t.Fatalf("branches %d/%d, want 1/2", branchCovered, branchTotal)
	}
}

func TestParseCoverageRejectsWhatIsNotAReport(t *testing.T) {
	for name, data := range map[string]string{
		"empty": "",
		"junit": `<testsuite name="x"><testcase name="y"/></testsuite>`,
		"text":  "hello",
	} {
		if _, err := ParseCoverage([]byte(data), "/work"); err == nil {
			t.Errorf("%s: want an error", name)
		} else if !strings.Contains(err.Error(), "report") && !strings.Contains(err.Error(), "empty") {
			t.Errorf("%s: error %q does not explain itself", name, err)
		}
	}
}
