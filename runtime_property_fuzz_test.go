package browsertester

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

const runtimePropertyDefaultCases = 128
const runtimePropertyDefaultSeed int64 = 1

const runtimeRerenderingFormHTML = `
<div id="mount"></div>
<script>
const mount = document.getElementById("mount");
const state = { name: "", checked: false, events: 0 };

function render() {
  mount.innerHTML =
    '<input id="name" value="' + state.name + '">' +
    '<input id="flag" type="checkbox" ' + (state.checked ? "checked" : "") + '>' +
    '<button id="commit">commit</button>' +
    '<p id="snapshot">' + state.name + '|' + state.checked + '|' + state.events + '</p>';

  const nameInput = document.getElementById("name");
  const flagInput = document.getElementById("flag");
  const commitButton = document.getElementById("commit");

  nameInput.addEventListener("input", () => {
    state.name = document.getElementById("name").value;
    state.events += 1;
    render();
  });

  flagInput.addEventListener("input", () => {
    state.checked = document.getElementById("flag").checked;
    state.events += 1;
    render();
  });

  commitButton.addEventListener("click", () => {
    state.events += 1;
    render();
  });
}

render();
</script>
`

type runtimePropertyActionKind int

const (
	runtimePropertyActionTypeText runtimePropertyActionKind = iota
	runtimePropertyActionSetChecked
	runtimePropertyActionClickCommit
	runtimePropertyActionFocusName
	runtimePropertyActionBlurName
)

type runtimePropertyAction struct {
	kind    runtimePropertyActionKind
	text    string
	checked bool
}

func TestRuntimeRerenderingFormActionsDoNotPanic(t *testing.T) {
	seed := runtimePropertySeed()
	cases := runtimePropertyCases()
	rng := rand.New(rand.NewSource(seed))

	for i := 0; i < cases; i++ {
		actions := runtimePropertyActionSequence(rng)
		t.Run(fmt.Sprintf("case-%03d", i), func(t *testing.T) {
			assertRuntimeRerenderingFormSequenceIsStable(t, seed, i, actions)
		})
	}
}

func assertRuntimeRerenderingFormSequenceIsStable(t *testing.T, seed int64, caseIndex int, actions []runtimePropertyAction) {
	t.Helper()

	harness := mustHarnessFromHTML(t, strings.TrimSpace(runtimeRerenderingFormHTML))
	wantedName := ""
	wantedChecked := false
	wantedEvents := 0

	for step, action := range actions {
		if err := runRuntimePropertyAction(harness, action); err != nil {
			t.Fatalf("seed=%d case=%d step=%d action=%+v error = %v", seed, caseIndex, step, action, err)
		}

		switch action.kind {
		case runtimePropertyActionTypeText:
			wantedName = action.text
			wantedEvents++
		case runtimePropertyActionSetChecked:
			wantedChecked = action.checked
			wantedEvents++
		case runtimePropertyActionClickCommit:
			wantedEvents++
		case runtimePropertyActionFocusName, runtimePropertyActionBlurName:
		default:
			t.Fatalf("seed=%d case=%d step=%d action=%+v has unknown kind", seed, caseIndex, step, action)
		}

		if err := harness.AssertExists("#name"); err != nil {
			t.Fatalf("seed=%d case=%d step=%d action=%+v AssertExists(#name) error = %v", seed, caseIndex, step, action, err)
		}
		if err := harness.AssertExists("#flag"); err != nil {
			t.Fatalf("seed=%d case=%d step=%d action=%+v AssertExists(#flag) error = %v", seed, caseIndex, step, action, err)
		}
		if err := harness.AssertExists("#commit"); err != nil {
			t.Fatalf("seed=%d case=%d step=%d action=%+v AssertExists(#commit) error = %v", seed, caseIndex, step, action, err)
		}
		if err := harness.AssertExists("#snapshot"); err != nil {
			t.Fatalf("seed=%d case=%d step=%d action=%+v AssertExists(#snapshot) error = %v", seed, caseIndex, step, action, err)
		}

		wantSnapshot := fmt.Sprintf("%s|%t|%d", wantedName, wantedChecked, wantedEvents)
		if err := harness.AssertText("#snapshot", wantSnapshot); err != nil {
			t.Fatalf("seed=%d case=%d step=%d action=%+v AssertText(#snapshot, %q) error = %v", seed, caseIndex, step, action, wantSnapshot, err)
		}
		if err := harness.AssertValue("#name", wantedName); err != nil {
			t.Fatalf("seed=%d case=%d step=%d action=%+v AssertValue(#name, %q) error = %v", seed, caseIndex, step, action, wantedName, err)
		}
		if err := harness.AssertChecked("#flag", wantedChecked); err != nil {
			t.Fatalf("seed=%d case=%d step=%d action=%+v AssertChecked(#flag, %t) error = %v", seed, caseIndex, step, action, wantedChecked, err)
		}
	}
}

func runRuntimePropertyAction(harness *Harness, action runtimePropertyAction) error {
	switch action.kind {
	case runtimePropertyActionTypeText:
		return harness.TypeText("#name", action.text)
	case runtimePropertyActionSetChecked:
		return harness.SetChecked("#flag", action.checked)
	case runtimePropertyActionClickCommit:
		return harness.Click("#commit")
	case runtimePropertyActionFocusName:
		return harness.Focus("#name")
	case runtimePropertyActionBlurName:
		return harness.Blur()
	default:
		return fmt.Errorf("unsupported runtime property action kind %d", action.kind)
	}
}

func runtimePropertyActionSequence(r *rand.Rand) []runtimePropertyAction {
	length := r.Intn(24) + 1
	actions := make([]runtimePropertyAction, length)
	for i := range actions {
		actions[i] = randomRuntimePropertyAction(r)
	}
	return actions
}

func randomRuntimePropertyAction(r *rand.Rand) runtimePropertyAction {
	switch r.Intn(12) {
	case 0, 1, 2, 3, 4:
		return runtimePropertyAction{
			kind: runtimePropertyActionTypeText,
			text: randomRuntimePropertyText(r),
		}
	case 5, 6, 7:
		return runtimePropertyAction{
			kind:    runtimePropertyActionSetChecked,
			checked: r.Intn(2) == 0,
		}
	case 8, 9:
		return runtimePropertyAction{kind: runtimePropertyActionClickCommit}
	case 10:
		return runtimePropertyAction{kind: runtimePropertyActionFocusName}
	default:
		return runtimePropertyAction{kind: runtimePropertyActionBlurName}
	}
}

func randomRuntimePropertyText(r *rand.Rand) string {
	const alphabet = "abcxyz0123 -_"
	length := r.Intn(11)
	out := make([]byte, length)
	for i := range out {
		out[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(out)
}

func runtimePropertyCases() int {
	return parsePositiveIntEnv("BROWSER_TESTER_RUNTIME_PROPTEST_CASES",
		parsePositiveIntEnv("BROWSER_TESTER_PROPTEST_CASES", runtimePropertyDefaultCases))
}

func runtimePropertySeed() int64 {
	return parseInt64Env("BROWSER_TESTER_RUNTIME_PROPTEST_SEED",
		parseInt64Env("BROWSER_TESTER_PROPTEST_SEED", runtimePropertyDefaultSeed))
}

func parseInt64Env(name string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func parsePositiveIntEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
