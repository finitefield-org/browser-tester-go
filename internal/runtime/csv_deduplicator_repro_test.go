package runtime

import (
	"strings"
	"testing"
)

func wrapScript(body string) string {
	body = strings.ReplaceAll(body, "__BT__", "`")
	return `<main><div id="out"></div><script>` + body + `</script></main>`
}

func TestCSVDeduplicatorEscapeHtmlParses(t *testing.T) {
	rawHTML := wrapScript(`const helper = { escapeHtml(value) { return String(value == null ? "" : value).replace(/&/g, "\u0026amp;").replace(/</g, "\u0026lt;").replace(/>/g, "\u0026gt;").replace(/\"/g, "&quot;").replace(/'/g, "\u0026#39;"); } }; document.getElementById("out").textContent = helper.escapeHtml(__BT__<tag>__BT__);`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() escapeHtml snippet error = %v", err)
	}
}

func TestCSVDeduplicatorBytesLabelParses(t *testing.T) {
	rawHTML := wrapScript(`const helper = { bytesLabel(size) { const n = Number(size); if (!Number.isFinite(n)) return "0 B"; if (n < 1024) return __BT__${n} B__BT__; const units = ["KB", "MB", "GB"]; let value = n / 1024; let unitIndex = 0; while (value >= 1024 && unitIndex < units.length - 1) { value /= 1024; unitIndex += 1; } return __BT__${value.toFixed(value >= 100 ? 0 : value >= 10 ? 1 : 2)} ${units[unitIndex]}__BT__; } }; document.getElementById("out").textContent = helper.bytesLabel(1536);`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() bytesLabel snippet error = %v", err)
	}
}

func TestCSVDeduplicatorParseInputTextParses(t *testing.T) {
	rawHTML := wrapScript(`function parseInputText() { try { const rows = [["a", "b"], ["c"]]; const mismatch = null; if (mismatch) { throw { code: "COLUMN_MISMATCH", line: 1 }; } const maxCols = rows.reduce((acc, row) => Math.max(acc, row.length), 0); const normalizedRows = rows.map((row) => { const out = row.slice(0, maxCols); while (out.length < maxCols) out.push(""); return out; }); let headers = []; headers = Array.from({ length: maxCols }, (_, index) => __BT__col${index + 1}__BT__); document.getElementById("out").textContent = headers.join("|") + "|" + normalizedRows.length; } catch (error) { document.getElementById("out").textContent = "err"; } } parseInputText();`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() parseInputText snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderResultTableParses(t *testing.T) {
	rawHTML := wrapScript(`function renderResultTable() { const headers = ["name", "email"]; const rows = [["Alpha", "a@x"], ["Beta", "b@x"]]; const head = __BT__<thead><tr>${headers.map((h) => __BT__<th>${h}</th>__BT__).join("")}</tr></thead>__BT__; const bodyRows = rows.map((row) => { return __BT__<tr>${headers.map((_, idx) => __BT__<td>${row[idx]}</td>__BT__).join("")}</tr>__BT__; }); document.getElementById("out").textContent = head + bodyRows.join(""); } renderResultTable();`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderResultTable snippet error = %v", err)
	}
}

func TestCSVDeduplicatorDecodeFileToTextParses(t *testing.T) {
	rawHTML := wrapScript(`async function decodeFileToText(file) { const buffer = await file.arrayBuffer(); const bytes = new Uint8Array(buffer); try { return new TextDecoder("utf-8", { fatal: true }).decode(bytes).replace(/^\uFEFF/, ""); } catch (_) { return new TextDecoder("shift_jis", { fatal: false }).decode(bytes).replace(/^\uFEFF/, ""); } }`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() decodeFileToText snippet error = %v", err)
	}
}

func TestCSVDeduplicatorDeduplicateRowsParses(t *testing.T) {
	rawHTML := wrapScript(`function deduplicateRows() {
  clearMessages();

  if (!state.parsed || !state.parsed.headers.length) {
    showError(page.Tool.ErrorNoInput);
    return;
  }

  if (!state.selectedKeyIndices.length) {
    showError(page.Tool.ErrorNoKey);
    return;
  }

  const settings = currentSettings();
  const rows = state.parsed.dataRows;
  const headers = state.parsed.headers;
  const rowLines = state.parsed.dataRowLines;

  const started = performance.now();

  const groupsByKey = new Map();
  const orderedKeys = [];

  rows.forEach((row, rowIndex) => {
    const normalizedParts = state.selectedKeyIndices.map((colIndex) => normalizeForKey(row[colIndex], settings));
    const displayParts = state.selectedKeyIndices.map((colIndex) => {
      const value = String(row[colIndex] == null ? "" : row[colIndex]);
      return settings.trimMode === "trim" ? value.trim() : value;
    });

    const allBlank = normalizedParts.every((part) => part === "");
    if (settings.blankKeyMode === "skip" && allBlank) {
      const key = __BT____blank__${rowIndex}__BT__;
      groupsByKey.set(key, {
        key,
        display: page.Tool.BlankKeyLabel,
        indices: [rowIndex],
        skippedBlank: true,
      });
      orderedKeys.push(key);
      return;
    }

    const key = normalizedParts.join("\u001F");
    if (!groupsByKey.has(key)) {
      groupsByKey.set(key, {
        key,
        display: displayParts.join(" + "),
        indices: [],
        skippedBlank: false,
      });
      orderedKeys.push(key);
    }

    groupsByKey.get(key).indices.push(rowIndex);
  });

  const dedupedRows = [];
  const duplicateRows = [];
  const groupRows = [];
  const topKeys = [];
  const conflicts = [];

  let duplicateGroups = 0;

  orderedKeys.forEach((key) => {
    const group = groupsByKey.get(key);
    if (!group) return;

    const size = group.indices.length;
    if (size <= 1) {
      dedupedRows.push(rows[group.indices[0]].slice());
      return;
    }

    if (!group.skippedBlank) {
      duplicateGroups += 1;
      groupRows.push([
        group.display || page.Tool.GroupKeyEmpty,
        String(size),
        group.indices.map((idx) => rowLines[idx] || idx + 1).join(","),
      ]);
      topKeys.push({ key: group.display || page.Tool.GroupKeyEmpty, count: size });
    }

    if (settings.rule === "first") {
      const keepIndex = group.indices[0];
      dedupedRows.push(rows[keepIndex].slice());
      group.indices.slice(1).forEach((idx) => duplicateRows.push(rows[idx].slice()));
      return;
    }

    if (settings.rule === "last") {
      const keepIndex = group.indices[group.indices.length - 1];
      dedupedRows.push(rows[keepIndex].slice());
      group.indices.forEach((idx) => {
        if (idx !== keepIndex) duplicateRows.push(rows[idx].slice());
      });
      return;
    }

    const merged = rows[group.indices[0]].slice();
    group.indices.slice(1).forEach((idx) => {
      const src = rows[idx];
      duplicateRows.push(src.slice());
      for (let col = 0; col < headers.length; col += 1) {
        const left = merged[col];
        const right = src[col];

        if (isBlank(left) && !isBlank(right)) {
          merged[col] = right;
          continue;
        }

        if (!isBlank(left) && !isBlank(right) && String(left) !== String(right)) {
          if (settings.conflictPolicy === "last") {
            merged[col] = right;
          }
          if (settings.conflictPolicy === "warn") {
            conflicts.push({
              key: group.display || page.Tool.GroupKeyEmpty,
              column: headers[col],
              left: String(left),
              right: String(right),
              rowLine: rowLines[idx] || idx + 1,
            });
          }
        }
      }
    });

    dedupedRows.push(merged);
  });

  topKeys.sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));

  const ended = performance.now();
  const summary = {
    originalRows: rows.length,
    resultRows: dedupedRows.length,
    removedRows: Math.max(0, rows.length - dedupedRows.length),
    duplicateGroups,
    processingMs: Math.round(ended - started),
    conflictCount: conflicts.length,
  };

  state.result = {
    headers: headers.slice(),
    dedupedRows,
    duplicateRows,
    groupRows,
    topKeys: topKeys.slice(0, 5),
    conflicts,
    summary,
    compareLine: helper.formatTemplate(page.Tool.CompareAppliedTemplate, {
      case: settings.caseMode === "ignore" ? page.Tool.CaseModeIgnore : page.Tool.CaseModeSensitive,
      rule: settings.rule === "first" ? page.Tool.RuleFirst : settings.rule === "last" ? page.Tool.RuleLast : page.Tool.RuleMerge,
      conflictPolicy: settings.conflictPolicy,
      normalized: settings.normalize ? page.Tool.Yes : page.Tool.No,
      blankKeyMode: settings.blankKeyMode,
    }),
  };

  renderResult();
  updateTargetHints();
  persistState();
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() deduplicateRows snippet error = %v", err)
	}
}

func TestCSVDeduplicatorDeduplicateRowsFollowedByFunctionParses(t *testing.T) {
	rawHTML := wrapScript(`function deduplicateRows() {
  clearMessages();

  if (!state.parsed || !state.parsed.headers.length) {
    showError(page.Tool.ErrorNoInput);
    return;
  }

  if (!state.selectedKeyIndices.length) {
    showError(page.Tool.ErrorNoKey);
    return;
  }

  const settings = currentSettings();
  const rows = state.parsed.dataRows;
  const headers = state.parsed.headers;
  const rowLines = state.parsed.dataRowLines;

  const started = performance.now();

  const groupsByKey = new Map();
  const orderedKeys = [];

  rows.forEach((row, rowIndex) => {
    const normalizedParts = state.selectedKeyIndices.map((colIndex) => normalizeForKey(row[colIndex], settings));
    const displayParts = state.selectedKeyIndices.map((colIndex) => {
      const value = String(row[colIndex] == null ? "" : row[colIndex]);
      return settings.trimMode === "trim" ? value.trim() : value;
    });

    const allBlank = normalizedParts.every((part) => part === "");
    if (settings.blankKeyMode === "skip" && allBlank) {
      const key = __BT____blank__${rowIndex}__BT__;
      groupsByKey.set(key, {
        key,
        display: page.Tool.BlankKeyLabel,
        indices: [rowIndex],
        skippedBlank: true,
      });
      orderedKeys.push(key);
      return;
    }

    const key = normalizedParts.join("\u001F");
    if (!groupsByKey.has(key)) {
      groupsByKey.set(key, {
        key,
        display: displayParts.join(" + "),
        indices: [],
        skippedBlank: false,
      });
      orderedKeys.push(key);
    }

    groupsByKey.get(key).indices.push(rowIndex);
  });

  const dedupedRows = [];
  const duplicateRows = [];
  const groupRows = [];
  const topKeys = [];
  const conflicts = [];

  let duplicateGroups = 0;

  orderedKeys.forEach((key) => {
    const group = groupsByKey.get(key);
    if (!group) return;

    const size = group.indices.length;
    if (size <= 1) {
      dedupedRows.push(rows[group.indices[0]].slice());
      return;
    }

    if (!group.skippedBlank) {
      duplicateGroups += 1;
      groupRows.push([
        group.display || page.Tool.GroupKeyEmpty,
        String(size),
        group.indices.map((idx) => rowLines[idx] || idx + 1).join(","),
      ]);
      topKeys.push({ key: group.display || page.Tool.GroupKeyEmpty, count: size });
    }

    if (settings.rule === "first") {
      const keepIndex = group.indices[0];
      dedupedRows.push(rows[keepIndex].slice());
      group.indices.slice(1).forEach((idx) => duplicateRows.push(rows[idx].slice()));
      return;
    }

    if (settings.rule === "last") {
      const keepIndex = group.indices[group.indices.length - 1];
      dedupedRows.push(rows[keepIndex].slice());
      group.indices.forEach((idx) => {
        if (idx !== keepIndex) duplicateRows.push(rows[idx].slice());
      });
      return;
    }

    const merged = rows[group.indices[0]].slice();
    group.indices.slice(1).forEach((idx) => {
      const src = rows[idx];
      duplicateRows.push(src.slice());
      for (let col = 0; col < headers.length; col += 1) {
        const left = merged[col];
        const right = src[col];

        if (isBlank(left) && !isBlank(right)) {
          merged[col] = right;
          continue;
        }

        if (!isBlank(left) && !isBlank(right) && String(left) !== String(right)) {
          if (settings.conflictPolicy === "last") {
            merged[col] = right;
          }
          if (settings.conflictPolicy === "warn") {
            conflicts.push({
              key: group.display || page.Tool.GroupKeyEmpty,
              column: headers[col],
              left: String(left),
              right: String(right),
              rowLine: rowLines[idx] || idx + 1,
            });
          }
        }
      }
    });

    dedupedRows.push(merged);
  });

  topKeys.sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));

  const ended = performance.now();
  const summary = {
    originalRows: rows.length,
    resultRows: dedupedRows.length,
    removedRows: Math.max(0, rows.length - dedupedRows.length),
    duplicateGroups,
    processingMs: Math.round(ended - started),
    conflictCount: conflicts.length,
  };

  state.result = {
    headers: headers.slice(),
    dedupedRows,
    duplicateRows,
    groupRows,
    topKeys: topKeys.slice(0, 5),
    conflicts,
    summary,
    compareLine: helper.formatTemplate(page.Tool.CompareAppliedTemplate, {
      case: settings.caseMode === "ignore" ? page.Tool.CaseModeIgnore : page.Tool.CaseModeSensitive,
      trim: settings.trimMode === "trim" ? page.Tool.TrimModeTrim : page.Tool.TrimModeKeep,
      normalize: settings.normalize ? page.Tool.NormalizeOn : page.Tool.NormalizeOff,
    }),
  };

  renderResult();

  el.status.textContent = helper.formatTemplate(page.Tool.StatusDoneTemplate, {
    original: numberFmt.format(summary.originalRows),
    result: numberFmt.format(summary.resultRows),
    removed: numberFmt.format(summary.removedRows),
    groups: numberFmt.format(summary.duplicateGroups),
    ms: numberFmt.format(summary.processingMs),
  });

  if (summary.duplicateGroups === 0) {
    showWarning(page.Tool.WarningNoDuplicates);
  }

  el.liveRegion.textContent = page.Tool.RunCompleted;
}
function renderResult() { return 1; }
document.getElementById("out").textContent = String(1);`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() deduplicateRows followed by function snippet error = %v", err)
	}
}

func TestCSVDeduplicatorBindEventsParses(t *testing.T) {
	rawHTML := wrapScript(`function bindEvents() {
  el.openButton.addEventListener("click", () => setDialogOpen(true));
  el.closeButton.addEventListener("click", () => setDialogOpen(false));

  el.dialog.addEventListener("click", (event) => {
    if (event.target === el.dialog) setDialogOpen(false);
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !el.dialog.classList.contains("hidden")) {
      setDialogOpen(false);
    }
    if ((event.ctrlKey || event.metaKey) && event.key === "Enter" && !el.dialog.classList.contains("hidden")) {
      event.preventDefault();
      deduplicateRows();
    }
  });

  el.settingsToggle.addEventListener("click", () => {
    el.advancedPanel.open = !el.advancedPanel.open;
    persistState();
  });

  el.inputModeButtons.forEach((button) => {
    button.addEventListener("click", () => setInputMode(button.dataset.inputMode));
  });

  el.fileSelectButton.addEventListener("click", () => el.fileInput.click());
  el.fileInput.addEventListener("change", async () => {
    const file = el.fileInput.files && el.fileInput.files[0];
    if (!file) return;
    try {
      const text = await decodeFileToText(file);
      state.fileText = text;
      el.fileMeta.textContent = helper.formatTemplate(page.Tool.FileMetaTemplate, {
        name: file.name,
        size: helper.bytesLabel(file.size),
      });
      el.fileMeta.classList.remove("hidden");
      parseInputText();
      persistState();
    } catch (_) {
      showError(page.Tool.ErrorDecode);
    }
  });

  el.dropzone.addEventListener("click", () => el.fileInput.click());
  el.dropzone.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      el.fileInput.click();
    }
  });
  el.dropzone.addEventListener("dragover", (event) => {
    event.preventDefault();
    el.dropzone.classList.add("dragover");
  });
  el.dropzone.addEventListener("dragleave", () => {
    el.dropzone.classList.remove("dragover");
  });
  el.dropzone.addEventListener("drop", async (event) => {
    event.preventDefault();
    el.dropzone.classList.remove("dragover");
    const file = event.dataTransfer && event.dataTransfer.files && event.dataTransfer.files[0];
    if (!file) return;
    try {
      const text = await decodeFileToText(file);
      state.fileText = text;
      el.fileMeta.textContent = helper.formatTemplate(page.Tool.FileMetaTemplate, {
        name: file.name,
        size: helper.bytesLabel(file.size),
      });
      el.fileMeta.classList.remove("hidden");
      parseInputText();
      persistState();
    } catch (_) {
      showError(page.Tool.ErrorDecode);
    }
  });

  el.pasteInput.addEventListener("input", handleInputChanged);

  el.sampleLoadButton.addEventListener("click", () => {
    state.sampleText = String(page.Tool.SampleData || "");
    el.samplePreview.textContent = state.sampleText;
    parseInputText();
  });

  el.hasHeader.addEventListener("change", handleInputChanged);
  el.delimiterMode.addEventListener("change", () => {
    renderDelimiterControls();
    handleInputChanged();
  });
  el.delimiterManual.addEventListener("change", handleInputChanged);
  el.encoding.addEventListener("change", handleInputChanged);

  el.keySearch.addEventListener("input", () => {
    state.keyFilter = el.keySearch.value || "";
    renderKeyList();
  });

  el.keyList.addEventListener("change", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLInputElement)) return;
    if (target.type !== "checkbox") return;
    const index = Number(target.dataset.keyIndex || "-1");
    if (!Number.isFinite(index) || index < 0) return;

    if (target.checked) {
      if (!state.selectedKeyIndices.includes(index)) state.selectedKeyIndices.push(index);
    } else {
      state.selectedKeyIndices = state.selectedKeyIndices.filter((item) => item !== index);
    }

    state.selectedKeyIndices.sort((a, b) => a - b);
    renderKeyChips();
    updateTargetHints();
    persistState();
  });

  el.keyChips.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;
    const indexText = target.dataset.keyRemoveIndex;
    if (!indexText) return;
    const index = Number(indexText);
    if (!Number.isFinite(index)) return;

    state.selectedKeyIndices = state.selectedKeyIndices.filter((item) => item !== index);
    renderKeyList();
    renderKeyChips();
    updateTargetHints();
    persistState();
  });

  el.ruleRadios.forEach((radio) => {
    radio.addEventListener("change", () => {
      renderConflictControls();
      persistState();
    });
  });

  el.conflictPolicy.addEventListener("change", persistState);
  el.caseRadios.forEach((radio) => radio.addEventListener("change", persistState));
  el.trimRadios.forEach((radio) => radio.addEventListener("change", persistState));
  el.blankKeyRadios.forEach((radio) => radio.addEventListener("change", persistState));
  el.normalize.addEventListener("change", persistState);

  el.runButton.addEventListener("click", deduplicateRows);
  el.mobileRunButton.addEventListener("click", deduplicateRows);

  el.clearInputButton.addEventListener("click", () => {
    if (state.inputMode === "file") {
      state.fileText = "";
      el.fileMeta.classList.add("hidden");
      el.fileMeta.textContent = "";
      el.fileInput.value = "";
    } else if (state.inputMode === "sample") {
      state.sampleText = "";
      el.samplePreview.textContent = "";
    } else {
      el.pasteInput.value = "";
    }
    parseInputText();
  });

  el.resetSampleButton.addEventListener("click", () => {
    state.sampleText = String(page.Tool.SampleData || "");
    el.samplePreview.textContent = state.sampleText;
    if (state.inputMode === "sample") {
      parseInputText();
    }
  });

  el.tabButtons.forEach((button) => {
    button.addEventListener("click", () => {
      state.activeResultTab = button.dataset.resultTab || "result";
      renderResultTable();
    });
  });

  el.downloadResult.addEventListener("click", downloadResultCsv);
  el.downloadDuplicates.addEventListener("click", downloadDuplicatesCsv);
  el.downloadGroups.addEventListener("click", downloadGroupsCsv);
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() bindEvents snippet error = %v", err)
	}
}

func TestCSVDeduplicatorBindEventsFollowedByFunctionParses(t *testing.T) {
	rawHTML := wrapScript(`function bindEvents() {
  el.openButton.addEventListener("click", () => setDialogOpen(true));
  el.closeButton.addEventListener("click", () => setDialogOpen(false));

  el.dialog.addEventListener("click", (event) => {
    if (event.target === el.dialog) setDialogOpen(false);
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !el.dialog.classList.contains("hidden")) {
      setDialogOpen(false);
    }
    if ((event.ctrlKey || event.metaKey) && event.key === "Enter" && !el.dialog.classList.contains("hidden")) {
      event.preventDefault();
      deduplicateRows();
    }
  });

  el.settingsToggle.addEventListener("click", () => {
    el.advancedPanel.open = !el.advancedPanel.open;
    persistState();
  });

  el.inputModeButtons.forEach((button) => {
    button.addEventListener("click", () => setInputMode(button.dataset.inputMode));
  });

  el.fileSelectButton.addEventListener("click", () => el.fileInput.click());
  el.fileInput.addEventListener("change", async () => {
    const file = el.fileInput.files && el.fileInput.files[0];
    if (!file) return;
    try {
      const text = await decodeFileToText(file);
      state.fileText = text;
      el.fileMeta.textContent = helper.formatTemplate(page.Tool.FileMetaTemplate, {
        name: file.name,
        size: helper.bytesLabel(file.size),
      });
      el.fileMeta.classList.remove("hidden");
      parseInputText();
      persistState();
    } catch (_) {
      showError(page.Tool.ErrorDecode);
    }
  });

  el.dropzone.addEventListener("click", () => el.fileInput.click());
  el.dropzone.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      el.fileInput.click();
    }
  });
  el.dropzone.addEventListener("dragover", (event) => {
    event.preventDefault();
    el.dropzone.classList.add("dragover");
  });
  el.dropzone.addEventListener("dragleave", () => {
    el.dropzone.classList.remove("dragover");
  });
  el.dropzone.addEventListener("drop", async (event) => {
    event.preventDefault();
    el.dropzone.classList.remove("dragover");
    const file = event.dataTransfer && event.dataTransfer.files && event.dataTransfer.files[0];
    if (!file) return;
    try {
      const text = await decodeFileToText(file);
      state.fileText = text;
      el.fileMeta.textContent = helper.formatTemplate(page.Tool.FileMetaTemplate, {
        name: file.name,
        size: helper.bytesLabel(file.size),
      });
      el.fileMeta.classList.remove("hidden");
      parseInputText();
      persistState();
    } catch (_) {
      showError(page.Tool.ErrorDecode);
    }
  });

  el.pasteInput.addEventListener("input", handleInputChanged);

  el.sampleLoadButton.addEventListener("click", () => {
    state.sampleText = String(page.Tool.SampleData || "");
    el.samplePreview.textContent = state.sampleText;
    parseInputText();
  });

  el.hasHeader.addEventListener("change", handleInputChanged);
  el.delimiterMode.addEventListener("change", () => {
    renderDelimiterControls();
    handleInputChanged();
  });
  el.delimiterManual.addEventListener("change", handleInputChanged);
  el.encoding.addEventListener("change", handleInputChanged);

  el.keySearch.addEventListener("input", () => {
    state.keyFilter = el.keySearch.value || "";
    renderKeyList();
  });

  el.keyList.addEventListener("change", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLInputElement)) return;
    if (target.type !== "checkbox") return;
    const index = Number(target.dataset.keyIndex || "-1");
    if (!Number.isFinite(index) || index < 0) return;

    if (target.checked) {
      if (!state.selectedKeyIndices.includes(index)) state.selectedKeyIndices.push(index);
    } else {
      state.selectedKeyIndices = state.selectedKeyIndices.filter((item) => item !== index);
    }

    state.selectedKeyIndices.sort((a, b) => a - b);
    renderKeyChips();
    updateTargetHints();
    persistState();
  });

  el.keyChips.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;
    const indexText = target.dataset.keyRemoveIndex;
    if (!indexText) return;
    const index = Number(indexText);
    if (!Number.isFinite(index)) return;

    state.selectedKeyIndices = state.selectedKeyIndices.filter((item) => item !== index);
    renderKeyList();
    renderKeyChips();
    updateTargetHints();
    persistState();
  });

  el.ruleRadios.forEach((radio) => {
    radio.addEventListener("change", () => {
      renderConflictControls();
      persistState();
    });
  });

  el.conflictPolicy.addEventListener("change", persistState);
  el.caseRadios.forEach((radio) => radio.addEventListener("change", persistState));
  el.trimRadios.forEach((radio) => radio.addEventListener("change", persistState));
  el.blankKeyRadios.forEach((radio) => radio.addEventListener("change", persistState));
  el.normalize.addEventListener("change", persistState);

  el.runButton.addEventListener("click", deduplicateRows);
  el.mobileRunButton.addEventListener("click", deduplicateRows);

  el.clearInputButton.addEventListener("click", () => {
    if (state.inputMode === "file") {
      state.fileText = "";
      el.fileMeta.classList.add("hidden");
      el.fileMeta.textContent = "";
      el.fileInput.value = "";
    } else if (state.inputMode === "sample") {
      state.sampleText = "";
      el.samplePreview.textContent = "";
    } else {
      el.pasteInput.value = "";
    }
    parseInputText();
  });

  el.resetSampleButton.addEventListener("click", () => {
    state.sampleText = String(page.Tool.SampleData || "");
    el.samplePreview.textContent = state.sampleText;
    if (state.inputMode === "sample") {
      parseInputText();
    }
  });

  el.tabButtons.forEach((button) => {
    button.addEventListener("click", () => {
      state.activeResultTab = button.dataset.resultTab || "result";
      renderResultTable();
    });
  });

  el.downloadResult.addEventListener("click", downloadResultCsv);
  el.downloadDuplicates.addEventListener("click", downloadDuplicatesCsv);
  el.downloadGroups.addEventListener("click", downloadGroupsCsv);
}
function afterBindEvents() { return 1; }
document.getElementById("out").textContent = String(afterBindEvents());`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() bindEvents followed by function snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderResultParses(t *testing.T) {
	rawHTML := wrapScript(`function renderResult() {
  const result = state.result;

  if (!result) {
    el.summaryOriginal.textContent = "0";
    el.summaryResult.textContent = "0";
    el.summaryRemoved.textContent = "0";
    el.summaryGroups.textContent = "0";
    el.compareLine.classList.add("hidden");
    el.resultTable.innerHTML = "";
    el.resultEmpty.textContent = page.Tool.EmptyState;
    el.resultEmpty.classList.remove("hidden");
    el.topKeys.innerHTML = __BT__<li>${helper.escapeHtml(page.Tool.TopKeysEmpty)}</li>__BT__;
    el.conflictsWrap.classList.add("hidden");
    setDownloadEnabled(false, false, false);
    syncTabButtons();
    return;
  }

  const summary = result.summary;
  el.summaryOriginal.textContent = numberFmt.format(summary.originalRows);
  el.summaryResult.textContent = numberFmt.format(summary.resultRows);
  el.summaryRemoved.textContent = numberFmt.format(summary.removedRows);
  el.summaryGroups.textContent = numberFmt.format(summary.duplicateGroups);

  el.compareLine.classList.remove("hidden");
  el.compareLine.textContent = result.compareLine;

  renderTopKeys(result.topKeys);
  renderConflicts(result.conflicts);
  renderResultTable();

  const hasResultRows = result.dedupedRows.length > 0;
  const hasDupRows = result.duplicateRows.length > 0;
  const hasGroups = result.groupRows.length > 0;
  setDownloadEnabled(hasResultRows, hasDupRows, hasGroups);
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderResult snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderResultFollowedByFunctionParses(t *testing.T) {
	rawHTML := wrapScript(`function renderResult() {
  const result = state.result;

  if (!result) {
    el.summaryOriginal.textContent = "0";
    el.summaryResult.textContent = "0";
    el.summaryRemoved.textContent = "0";
    el.summaryGroups.textContent = "0";
    el.compareLine.classList.add("hidden");
    el.resultTable.innerHTML = "";
    el.resultEmpty.textContent = page.Tool.EmptyState;
    el.resultEmpty.classList.remove("hidden");
    el.topKeys.innerHTML = __BT__<li>${helper.escapeHtml(page.Tool.TopKeysEmpty)}</li>__BT__;
    el.conflictsWrap.classList.add("hidden");
    setDownloadEnabled(false, false, false);
    syncTabButtons();
    return;
  }

  const summary = result.summary;
  el.summaryOriginal.textContent = numberFmt.format(summary.originalRows);
  el.summaryResult.textContent = numberFmt.format(summary.resultRows);
  el.summaryRemoved.textContent = numberFmt.format(summary.removedRows);
  el.summaryGroups.textContent = numberFmt.format(summary.duplicateGroups);

  el.compareLine.classList.remove("hidden");
  el.compareLine.textContent = result.compareLine;

  renderTopKeys(result.topKeys);
  renderConflicts(result.conflicts);
  renderResultTable();

  const hasResultRows = result.dedupedRows.length > 0;
  const hasDupRows = result.duplicateRows.length > 0;
  const hasGroups = result.groupRows.length > 0;
  setDownloadEnabled(hasResultRows, hasDupRows, hasGroups);
}
function afterRenderResult() { return 1; }
document.getElementById("out").textContent = String(afterRenderResult());`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderResult followed by function snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderTableParses(t *testing.T) {
	rawHTML := wrapScript(`function renderTable(table, headers, rows) {
  if (!table) return;
  const head = __BT__<thead><tr>${headers.map((h) => __BT__<th>${helper.escapeHtml(h)}</th>__BT__).join("")}</tr></thead>__BT__;
  const bodyRows = rows.map((row) => {
    return __BT__<tr>${headers.map((_, idx) => __BT__<td>${helper.escapeHtml(row[idx])}</td>__BT__).join("")}</tr>__BT__;
  }).join("");
  table.innerHTML = __BT__${head}<tbody>${bodyRows}</tbody>__BT__;
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderTable snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderTableFollowedByFunctionParses(t *testing.T) {
	rawHTML := wrapScript(`function renderTable(table, headers, rows) {
  if (!table) return;
  const head = __BT__<thead><tr>${headers.map((h) => __BT__<th>${helper.escapeHtml(h)}</th>__BT__).join("")}</tr></thead>__BT__;
  const bodyRows = rows.map((row) => {
    return __BT__<tr>${headers.map((_, idx) => __BT__<td>${helper.escapeHtml(row[idx])}</td>__BT__).join("")}</tr>__BT__;
  }).join("");
  table.innerHTML = __BT__${head}<tbody>${bodyRows}</tbody>__BT__;
}
function afterRenderTable() { return 1; }
document.getElementById("out").textContent = String(afterRenderTable());`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderTable followed by function snippet error = %v", err)
	}
}

func TestCSVDeduplicatorToCsvParses(t *testing.T) {
	rawHTML := wrapScript(`function toCsv(headers, rows) {
  const lines = [];
  lines.push(headers.map(csvEscape).join(","));
  rows.forEach((row) => {
    lines.push(headers.map((_, idx) => csvEscape(row[idx])).join(","));
  });
  return __BT__${lines.join("\r\n")}\r\n__BT__;
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() toCsv snippet error = %v", err)
	}
}

func TestCSVDeduplicatorToCsvFollowedByFunctionParses(t *testing.T) {
	rawHTML := wrapScript(`function toCsv(headers, rows) {
  const lines = [];
  lines.push(headers.map(csvEscape).join(","));
  rows.forEach((row) => {
    lines.push(headers.map((_, idx) => csvEscape(row[idx])).join(","));
  });
  return __BT__${lines.join("\r\n")}\r\n__BT__;
}
function afterToCsv() { return 1; }
document.getElementById("out").textContent = String(afterToCsv());`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() toCsv followed by function snippet error = %v", err)
	}
}

func TestCSVDeduplicatorMeasureDelimiterParses(t *testing.T) {
	rawHTML := wrapScript(`function measureDelimiter(text, delimiter) {
  const maxChars = Math.min(text.length, 65536);
  const counts = [];
  let inQuotes = false;
  let fields = 1;
  let occurrences = 0;
  let rows = 0;

  for (let i = 0; i < maxChars; i += 1) {
    const ch = text[i];

    if (inQuotes) {
      if (ch === '"') {
        if (text[i + 1] === '"') i += 1;
        else inQuotes = false;
      }
      continue;
    }

    if (ch === '"') {
      inQuotes = true;
      continue;
    }

    if (ch === delimiter) {
      occurrences += 1;
      fields += 1;
      continue;
    }

    if (ch === "\r" || ch === "\n") {
      if (ch === "\r" && text[i + 1] === "\n") i += 1;
      counts.push(fields);
      fields = 1;
      rows += 1;
      if (rows >= 200) break;
    }
  }

  if (counts.length === 0) counts.push(fields);

  const avg = counts.reduce((sum, value) => sum + value, 0) / counts.length;
  const variance = counts.reduce((sum, value) => {
    const diff = value - avg;
    return sum + diff * diff;
  }, 0) / counts.length;

  let score = occurrences - variance * 5;
  if (avg < 2) score -= 8;

  return { delimiter, score, avgCols: avg, occurrences };
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() measureDelimiter snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderInputPreviewParses(t *testing.T) {
	rawHTML := wrapScript(`function renderInputPreview() {
  if (!state.parsed || !state.parsed.headers.length) {
    el.inputPreviewTable.innerHTML = "";
    el.inputPreviewNote.classList.add("hidden");
    return;
  }
  const rows = state.parsed.dataRows.slice(0, 50);
  renderTable(el.inputPreviewTable, state.parsed.headers, rows);
  el.inputPreviewNote.classList.remove("hidden");
  el.inputPreviewNote.textContent = page.Tool.InputPreviewLimit;
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderInputPreview snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderKeyListParses(t *testing.T) {
	rawHTML := wrapScript(`function renderKeyList() {
  el.keyList.innerHTML = "";
  if (!state.parsed || !state.parsed.headers.length) {
    const line = document.createElement("p");
    line.className = "text-xs tool-text-muted";
    line.textContent = page.Tool.KeyListEmpty;
    el.keyList.appendChild(line);
    return;
  }

  const filter = String(state.keyFilter || "").trim().toLowerCase();
  const headers = state.parsed.headers;

  headers.forEach((header, index) => {
    if (filter && !header.toLowerCase().includes(filter)) return;

    const label = document.createElement("label");
    label.className = "key-item";

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.className = "h-4 w-4";
    checkbox.checked = state.selectedKeyIndices.includes(index);
    checkbox.dataset.keyIndex = String(index);

    const span = document.createElement("span");
    span.textContent = header;

    label.appendChild(checkbox);
    label.appendChild(span);
    el.keyList.appendChild(label);
  });

  if (!el.keyList.children.length) {
    const line = document.createElement("p");
    line.className = "text-xs tool-text-muted";
    line.textContent = page.Tool.KeyFilterNoMatch;
    el.keyList.appendChild(line);
  }
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderKeyList snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderKeyChipsParses(t *testing.T) {
	rawHTML := wrapScript(`function renderKeyChips() {
  el.keyChips.innerHTML = "";
  if (!state.parsed || !state.selectedKeyIndices.length) {
    const span = document.createElement("span");
    span.className = "text-xs tool-text-muted";
    span.textContent = page.Tool.KeySelectedNone;
    el.keyChips.appendChild(span);
    return;
  }

  const headers = state.parsed.headers;
  state.selectedKeyIndices.forEach((index) => {
    if (index < 0 || index >= headers.length) return;
    const chip = document.createElement("span");
    chip.className = "chip";

    const text = document.createElement("span");
    text.textContent = headers[index];

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "chip-remove";
    remove.dataset.keyRemoveIndex = String(index);
    remove.setAttribute("aria-label", page.Tool.KeyChipRemoveAria);
    remove.textContent = "x";

    chip.appendChild(text);
    chip.appendChild(remove);
    el.keyChips.appendChild(chip);
  });
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderKeyChips snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderTopKeysParses(t *testing.T) {
	rawHTML := wrapScript(`function renderTopKeys(topKeys) {
  el.topKeys.innerHTML = "";
  if (!topKeys || !topKeys.length) {
    const li = document.createElement("li");
    li.textContent = page.Tool.TopKeysEmpty;
    el.topKeys.appendChild(li);
    return;
  }

  topKeys.forEach((item) => {
    const li = document.createElement("li");
    li.textContent = helper.formatTemplate(page.Tool.TopKeyItemTemplate, {
      key: item.key,
      count: numberFmt.format(item.count),
    });
    el.topKeys.appendChild(li);
  });
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderTopKeys snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRenderConflictsParses(t *testing.T) {
	rawHTML := wrapScript(`function renderConflicts(conflicts) {
  el.conflicts.innerHTML = "";
  if (!conflicts || !conflicts.length) {
    el.conflictsWrap.classList.add("hidden");
    return;
  }

  conflicts.forEach((conflict) => {
    const li = document.createElement("li");
    li.textContent = helper.formatTemplate(page.Tool.ConflictItemTemplate, {
      key: conflict.key,
      column: conflict.column,
      left: conflict.left,
      right: conflict.right,
      line: numberFmt.format(conflict.rowLine || 1),
    });
    el.conflicts.appendChild(li);
  });

  el.conflictsWrap.classList.remove("hidden");
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() renderConflicts snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRowIsBlankParses(t *testing.T) {
	rawHTML := wrapScript(`function rowIsBlank(row) {
  if (!Array.isArray(row) || row.length === 0) return true;
  return row.every((cell) => String(cell == null ? "" : cell) === "");
}
document.getElementById("out").textContent = String(rowIsBlank([]));`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() rowIsBlank snippet error = %v", err)
	}
}

func TestCSVDeduplicatorNormalizeRowsThenRowIsBlankParses(t *testing.T) {
	rawHTML := wrapScript(`function normalizeRows(rows, maxCols) {
  return rows.map((row) => {
    const out = row.slice(0, maxCols);
    while (out.length < maxCols) out.push("");
    return out;
  });
}
function rowIsBlank(row) {
  if (!Array.isArray(row) || row.length === 0) return true;
  return row.every((cell) => String(cell == null ? "" : cell) === "");
}
document.getElementById("out").textContent = String(rowIsBlank(normalizeRows([[1]], 1)[0]));`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() normalizeRows+rowIsBlank snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRowIsBlankFollowedByStatementParses(t *testing.T) {
	rawHTML := wrapScript(`function rowIsBlank(row) {
  if (!Array.isArray(row) || row.length === 0) return true;
  return row.every((cell) => String(cell == null ? "" : cell) === "");
}
const after = 1;
document.getElementById("out").textContent = String(after);`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() rowIsBlank followed by statement snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRowIsBlankFollowedByFunctionParses(t *testing.T) {
	rawHTML := wrapScript(`function rowIsBlank(row) {
  if (!Array.isArray(row) || row.length === 0) return true;
  return row.every((cell) => String(cell == null ? "" : cell) === "");
}
function findMismatch(rows) {
  if (!rows.length) return null;
  return rows.length;
}
document.getElementById("out").textContent = String(findMismatch([]));`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() rowIsBlank followed by function snippet error = %v", err)
	}
}

func TestCSVDeduplicatorRowIsBlankExactSourceParses(t *testing.T) {
	rawHTML := wrapScript(`function rowIsBlank(row) {
  if (!Array.isArray(row) || row.length === 0) return true;
  return row.every((cell) => String(cell == null ? "" : cell) === "");
}
function afterRowIsBlank() { return 1; }
document.getElementById("out").textContent = String(afterRowIsBlank());`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() rowIsBlank exact-source snippet error = %v", err)
	}
}

func TestCSVDeduplicatorSyncTabButtonsParses(t *testing.T) {
	rawHTML := wrapScript(`function syncTabButtons() {
  el.tabButtons.forEach((button) => {
    const active = button.dataset.resultTab === state.activeResultTab;
    button.classList.toggle("active", active);
    button.setAttribute("aria-selected", active ? "true" : "false");
  });
}`)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() syncTabButtons snippet error = %v", err)
	}
}

func TestCSVDeduplicatorHeaderNotesIIFEParses(t *testing.T) {
	rawHTML := `<main><div id="out"></div><script>if (false) { (function () { function hasClass(element, className) { return !!(element && element.classList && element.classList.contains(className)); } function isTitleTag(tagName) { return tagName === "H1" || tagName === "H2" || tagName === "H3" || tagName === "H4"; } function isHeaderNote(element) { if (!element || element.nodeType !== 1) { return false; } if (element.tagName === "P") { return true; } if (hasClass(element, "chip")) { return true; } if (hasClass(element, "trust-line") || hasClass(element, "privacy-line")) { return true; } if (hasClass(element, "trust-grid") || hasClass(element, "dialog-subtitle")) { return true; } if (hasClass(element, "disclaimer-line")) { return true; } if (hasClass(element, "line") && hasClass(element, "privacy")) { return true; } return hasClass(element, "text-xs") && hasClass(element, "tool-text-muted"); } function hideTitleCompanionNotes(container) { if (!container || !container.children) { return; } const children = Array.from(container.children); children.forEach((child, index) => { if (!isTitleTag(child.tagName)) { return; } let cursor = children[index + 1]; let cursorIndex = index + 1; while (cursor && isHeaderNote(cursor)) { cursor.remove(); cursorIndex += 1; cursor = children[cursorIndex]; } }); } function removeLeadingNotes(container) { if (!container || !container.children) { return; } let cursor = container.firstElementChild; while (cursor && isHeaderNote(cursor)) { cursor.remove(); cursor = cursor.nextElementSibling; } if (hasClass(container, "mode-bar")) { Array.from(container.children).forEach((child) => { if (isHeaderNote(child)) { child.remove(); } }); } } function hideDialogNotes() { document.querySelectorAll('[role="dialog"]').forEach((dialog) => { const panel = dialog.firstElementChild; if (!panel || !panel.children.length) { return; } const panelChildren = Array.from(panel.children); const header = panelChildren[0]; hideTitleCompanionNotes(header); Array.from(header.children).forEach((child) => { hideTitleCompanionNotes(child); }); panelChildren.slice(1).forEach((child) => { removeLeadingNotes(child); }); }); } if (document.readyState === "loading") { document.addEventListener("DOMContentLoaded", hideDialogNotes); return; } hideDialogNotes(); })(); }</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() header notes IIFE snippet error = %v", err)
	}
}

func TestCSVDeduplicatorServerCtaPlacementsScriptParses(t *testing.T) {
	rawHTML := `<main><div id="out"></div><script>if (false) { (function ensureServerCtaPlacements() { const path = window.location && window.location.pathname ? window.location.pathname : ""; const normalizeServerPath = (rawPath) => { if (!rawPath) return ""; return rawPath.replace(/\/index\.html$/i, "").replace(/\/+$/, ""); }; const isServerDetailPage = (value) => { if (!value) return false; if (value === "/server") return false; return /^\/(?:[a-z]{2,5}\/)?(?:server\/)?[^/]+$/.test(value); }; const normalizedPath = normalizeServerPath(path); if (!isServerDetailPage(normalizedPath)) return; const root = document.getElementById("hostinger-content") || document.querySelector("[data-article-content]") || document.querySelector("main"); if (!root) return; const links = Array.from(document.querySelectorAll("a[href]")); links.forEach((link) => { const href = link.getAttribute("href") || ""; if (!href || href[0] !== "/") return; if (href.startsWith("mailto:") || href.startsWith("tel:")) return; if (href === "/server" || href === "/server/") return; const normalizedHref = normalizeServerPath(href); if (!isServerDetailPage(normalizedHref)) return; const text = (link.textContent || "").trim(); if (!text) return; const cta = document.createElement("a"); cta.textContent = text; cta.href = href; cta.className = "server-cta-link"; cta.setAttribute("aria-label", text); link.replaceWith(cta); }); })(); }</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() server cta placements snippet error = %v", err)
	}
}
