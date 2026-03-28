package browsertester

import "testing"

func TestIssue202AsyncDigestStubUpdatesDomAfterAwait(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<div id="err"></div>
		<div id="meta"></div>
		<script>
		  if (!window.crypto) { window.crypto = {}; }
		  window.crypto.subtle = {
		    digest: function (_alg, _data) {
		      return Promise.resolve(new Uint8Array([65, 66, 67]).buffer);
		    }
		  };

		  (async function () {
		    const digest = await crypto.subtle.digest("SHA-256", new Uint8Array([1, 2, 3]));
		    document.getElementById("meta").textContent =
		      typeof digest + ":" + String(digest && digest.byteLength);
		    document.getElementById("out").textContent =
		      Array.from(new Uint8Array(digest)).join(",");
		  })().catch(function (error) {
		    document.getElementById("err").textContent =
		      error && error.message ? error.message : String(error);
		  });
		</script>
	`)

	if err := harness.AssertText("#err", ""); err != nil {
		t.Fatalf("AssertText(#err, \"\") error = %v", err)
	}
	if err := harness.AssertText("#meta", "object:3"); err != nil {
		t.Fatalf("AssertText(#meta, object:3) error = %v", err)
	}
	if err := harness.AssertText("#out", "65,66,67"); err != nil {
		t.Fatalf("AssertText(#out, 65,66,67) error = %v", err)
	}
}

func TestIssue202WindowPropertyReadsAsGlobalIdentifierInsideFunction(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  function installAndRead() {
		    window.hashApi = { tag: "ok" };
		    document.getElementById("out").textContent =
		      typeof hashApi + ":" + hashApi.tag;
		  }

		  installAndRead();
		</script>
	`)

	if err := harness.AssertText("#out", "object:ok"); err != nil {
		t.Fatalf("AssertText(#out, object:ok) error = %v", err)
	}
}
