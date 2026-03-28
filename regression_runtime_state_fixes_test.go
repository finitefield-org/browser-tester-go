package browsertester

import "testing"

func TestRegressionRuntimeStateFixesRecursiveConstArrowClosureCanReferenceItself(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id='run'>run</button>
		<p id='out'></p>
		<script>
		  const choose = (arr, k) => {
		    const out = [];
		    const recur = (start, cur) => {
		      if (cur.length === k) {
		        out.push([...cur]);
		        return;
		      }
		      for (let i = start; i < arr.length; i += 1) {
		        cur.push(arr[i]);
		        recur(i + 1, cur);
		        cur.pop();
		      }
		    };
		    recur(0, []);
		    return out;
		  };

		  document.getElementById('run').addEventListener('click', () => {
		    const combos = choose([1, 2, 3], 2);
		    document.getElementById('out').textContent = String(combos.length);
		  });
		</script>
	`)

	if err := harness.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}
	if err := harness.AssertText("#out", "3"); err != nil {
		t.Fatalf("AssertText(#out, 3) error = %v", err)
	}
}
