package browsertester

import "testing"

func TestIssue203StickyElementStaysPinnedAfterWindowScroll(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<style>
		  body { margin: 0; }
		  #sticky {
		    position: sticky;
		    top: 0;
		    height: 40px;
		    background: #ddd;
		  }
		  #spacer {
		    height: 1600px;
		  }
		</style>
		<div id="sticky">sticky</div>
		<div id="spacer"></div>
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  const sticky = document.getElementById("sticky");
		  const out = document.getElementById("out");
		  document.getElementById("go").addEventListener("click", () => {
		    const beforeTop = Math.round(sticky.getBoundingClientRect().top);
		    window.scrollTo(0, 300);
		    const afterTop = Math.round(sticky.getBoundingClientRect().top);
		    out.textContent =
		      "scrollY=" + window.scrollY + ",before=" + beforeTop + ",after=" + afterTop;
		  });
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "scrollY=300,before=0,after=0"); err != nil {
		t.Fatalf("AssertText(#out, scrollY=300,before=0,after=0) error = %v", err)
	}
}

func TestIssue203StickyElementHonorsRemTopInsetDuringScroll(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<style>
		  body { margin: 0; }
		  #sticky {
		    position: sticky;
		    top: 5.75rem;
		    height: 40px;
		    background: #ddd;
		  }
		  #spacer {
		    height: 1600px;
		  }
		</style>
		<div id="sticky">sticky</div>
		<div id="spacer"></div>
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  const sticky = document.getElementById("sticky");
		  const out = document.getElementById("out");
		  document.getElementById("go").addEventListener("click", () => {
		    const beforeTop = Math.round(sticky.getBoundingClientRect().top);
		    window.scrollTo(0, 300);
		    const afterTop = Math.round(sticky.getBoundingClientRect().top);
		    out.textContent =
		      "scrollY=" + window.scrollY + ",before=" + beforeTop + ",after=" + afterTop;
		  });
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "scrollY=300,before=92,after=92"); err != nil {
		t.Fatalf("AssertText(#out, scrollY=300,before=92,after=92) error = %v", err)
	}
}
