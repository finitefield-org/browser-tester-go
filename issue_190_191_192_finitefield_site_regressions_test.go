package browsertester

import "testing"

func TestIssue190DocumentActiveElementTagNameIsSupported(t *testing.T) {
	harness, err := FromHTML(`
      <textarea id="field"></textarea>
      <div id="out"></div>
      <script>
        const field = document.getElementById("field");
        field.focus();
        document.getElementById("out").textContent = document.activeElement.tagName;
      </script>
    `)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "TEXTAREA"); err != nil {
		t.Fatalf("AssertText(#out, TEXTAREA) error = %v", err)
	}
}

func TestIssue191DataUrlAnchorDownloadIsCapturedAsArtifact(t *testing.T) {
	harness, err := FromHTML(`
      <body>
        <button id="download">Download</button>
        <script>
          document.getElementById("download").addEventListener("click", () => {
            const link = document.createElement("a");
            link.href = "data:text/csv;charset=utf-8,%EF%BB%BFa%2Cb%0A1%2C2";
            link.download = "sample.csv";
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
          });
        </script>
      </body>
    `)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#download"); err != nil {
		t.Fatalf("Click(#download) error = %v", err)
	}
	downloads := harness.Mocks().Downloads().Artifacts()
	if len(downloads) != 1 {
		t.Fatalf("Downloads().Artifacts() = %#v, want one captured download", downloads)
	}
	if downloads[0].FileName != "sample.csv" {
		t.Fatalf("Downloads()[0].FileName = %q, want sample.csv", downloads[0].FileName)
	}
	if got, want := string(downloads[0].Bytes), "\ufeffa,b\n1,2"; got != want {
		t.Fatalf("Downloads()[0].Bytes = %q, want %q", got, want)
	}
}

func TestIssue192ArrayFlatIsSupported(t *testing.T) {
	harness, err := FromHTML(`
      <div id="out"></div>
      <script>
        const values = [["north"], ["south"]].flat();
        document.getElementById("out").textContent = values.join(",");
      </script>
    `)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "north,south"); err != nil {
		t.Fatalf("AssertText(#out, north,south) error = %v", err)
	}
}

func TestIssue192ArrayFlatHonorsDepthAndSkipsSparseSlots(t *testing.T) {
	harness, err := FromHTML(`
      <div id="out"></div>
      <script>
        let nested = [];
        nested[0] = 1;
        nested[1] = [2, [3]];
        nested[2] = "skip";
        delete nested[2];
        nested[3] = [4];
        const result = nested.flat(2);
        document.getElementById("out").textContent = result.join(",");
      </script>
    `)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "1,2,3,4"); err != nil {
		t.Fatalf("AssertText(#out, 1,2,3,4) error = %v", err)
	}
}
