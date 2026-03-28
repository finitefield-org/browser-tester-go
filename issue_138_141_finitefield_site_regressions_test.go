package browsertester

import "testing"

func TestIssue138GenericFormatTwoArgsIsNotHijackedAsIntlRelativeTime(t *testing.T) {
	harness, err := FromHTML(`
		<p id='out'></p>
		<script>
		  const helper = {
		    format(template, values) {
		      return template
		        .replace('{shown}', String(values.shown))
		        .replace('{total}', String(values.total));
		    }
		  };

		  function setStatus(text) {
		    document.getElementById('out').textContent = text;
		  }

		  const shown = 3;
		  const total = 8;
		  setStatus(helper.format('Shown {shown}/{total}', { shown, total }));
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "Shown 3/8" {
		t.Fatalf("TextContent(#out) = %q, want Shown 3/8", got)
	}
}

func TestIssue139FunctionCanReferenceLaterDeclaredConst(t *testing.T) {
	harness, err := FromHTML(`
		<div id='out'></div>
		<script>
		  function ensure() {
		    return state.value;
		  }

		  const state = { value: 123 };
		  document.getElementById('out').textContent = String(ensure());
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "123" {
		t.Fatalf("TextContent(#out) = %q, want 123", got)
	}
}

func TestIssue140NestedStatePathsAreNotTreatedAsDomElementVariables(t *testing.T) {
	harness, err := FromHTML(`
		<button id='btn'>run</button>
		<p id='out'></p>
		<script>
		  const state = {
		    ratio: { mode: 'a' },
		    measurements: [{ value: 1 }],
		  };

		  document.getElementById('btn').addEventListener('click', () => {
		    state.ratio.mode = 'b';
		    state.measurements[0].value = 2;
		    document.getElementById('out').textContent = state.ratio.mode + ':' + String(state.measurements[0].value);
		  });
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}
	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "b:2" {
		t.Fatalf("TextContent(#out) = %q, want b:2", got)
	}
}

func TestIssue141DispatchKeyboardBubblesToDelegatedListener(t *testing.T) {
	harness, err := FromHTML(`
		<div id='root'>
		  <input id='field'>
		</div>
		<p id='out'></p>
		<script>
		  document.getElementById('root').addEventListener('keydown', (event) => {
		    document.getElementById('out').textContent = 'root:' + event.target.id + ':' + event.key;
		  });
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.DispatchKeyboard("#field"); err != nil {
		t.Fatalf("DispatchKeyboard(#field) error = %v", err)
	}
	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "root:field:Escape" {
		t.Fatalf("TextContent(#out) = %q, want root:field:Escape", got)
	}
}
