const appsEl = document.getElementById("apps");
const machineEl = document.getElementById("machine");
const noticeEl = document.getElementById("notice");
const searchAppsEl = document.getElementById("searchApps");
const navStatusEl = document.getElementById("navStatus");
const jobStateEl = document.getElementById("jobState");
const jobLogEl = document.getElementById("jobLog");
const barFillEl = document.getElementById("barFill");
const percentEl = document.getElementById("percent");
const currentAppEl = document.getElementById("currentApp");
const stageEl = document.getElementById("stage");
const fileCountEl = document.getElementById("fileCount");
const byteCountEl = document.getElementById("byteCount");
const stickyProgressEl = document.getElementById("stickyProgress");
const stickyTitleEl = document.getElementById("stickyTitle");
const stickyDetailEl = document.getElementById("stickyDetail");
const stickyPercentEl = document.getElementById("stickyPercent");
const stickyBarFillEl = document.getElementById("stickyBarFill");
const selectedCountEl = document.getElementById("selectedCount");
const selectedListEl = document.getElementById("selectedList");
let apps = [];
let appChecks = {};
let completedApps = {};
let selectedAppNames = new Set();
let currentJob = null;
let pollTimer = null;
let hideProgressTimer = null;

async function api(path, options) {
  const res = await fetch(path, options);
  const data = await res.json();
  if (!res.ok || data.success === false) {
    const msg = data.error && data.error.message ? data.error.message : res.statusText;
    throw new Error(msg);
  }
  return data.data || data;
}

async function loadStatus() {
  const status = await api("/api/local/status");
  machineEl.textContent = `Computer: ${status.hostname || "-"} | Install: ${status.install_path}`;
}

async function loadApps() {
  noticeEl.textContent = "Checking applications...";
  try {
    const data = await api("/api/apps");
    apps = data.apps || [];
    renderApps();
    noticeEl.textContent = `${apps.length} application(s) found.`;
    navStatusEl.textContent = `${apps.length} apps`;
  } catch (err) {
    appsEl.innerHTML = "";
    noticeEl.textContent = err.message;
    navStatusEl.textContent = "Server error";
  }
}

function renderApps() {
  const query = searchAppsEl.value.trim().toLowerCase();
  const visibleApps = query
    ? apps.filter(app => `${app.name} ${app.path || ""}`.toLowerCase().includes(query))
    : apps;
  appsEl.innerHTML = visibleApps.map(app => `
    <article class="app-card ${completedApps[app.name] ? "updated" : ""}" id="card-${safeId(app.name)}">
      <label class="app-title">
        <input type="checkbox" value="${escapeHtml(app.name)}" ${selectedAppNames.has(app.name) ? "checked" : ""}>
        <span class="app-name">${escapeHtml(app.name)}</span>
        <strong class="updated-mark" id="mark-${safeId(app.name)}">${completedApps[app.name] ? "Done" : ""}</strong>
      </label>
      <div class="app-meta">${escapeHtml(app.path || app.name)}</div>
      <div class="check-result" id="check-${safeId(app.name)}">${renderCheck(app.name)}</div>
      <div class="app-actions">
        <button class="secondary" data-check="${escapeHtml(app.name)}">Up Check</button>
        <button class="primary" data-update="${escapeHtml(app.name)}">Update</button>
      </div>
    </article>
  `).join("");
  if (!visibleApps.length) {
    appsEl.innerHTML = `<div class="empty-state">No programs found.</div>`;
  }
  updateSelectedSummary();
}

function renderCheck(appName) {
  const check = appChecks[appName];
  if (!check) return "Not checked";
  if (check.status === "checking") return "Checking...";
  if (check.status === "failed") return `Check failed: ${escapeHtml(check.error)}`;
  return `${check.file_count} files | ${formatBytes(check.total_bytes)}`;
}

async function upCheck(appName) {
  appChecks[appName] = { status: "checking" };
  updateCheckResult(appName);
  try {
    const data = await api(`/api/apps/${encodeURIComponent(appName)}/files`);
    appChecks[appName] = {
      status: "success",
      file_count: data.file_count,
      total_bytes: data.total_bytes
    };
    noticeEl.textContent = `${appName}: ${data.file_count} files, ${formatBytes(data.total_bytes)} ready.`;
  } catch (err) {
    appChecks[appName] = { status: "failed", error: err.message };
    noticeEl.textContent = `${appName}: ${err.message}`;
  }
  updateCheckResult(appName);
}

function updateCheckResult(appName) {
  const el = document.getElementById(`check-${safeId(appName)}`);
  if (el) el.innerHTML = renderCheck(appName);
}

async function startUpdate(selected) {
  if (!selected.length) {
    noticeEl.textContent = "Select at least one application.";
    return;
  }
  setButtons(true);
  clearTimeout(hideProgressTimer);
  showStickyProgress("Starting update", selected.join(", "), 0);
  try {
    const data = await api("/api/update", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ applications: selected })
    });
    currentJob = data.job_id;
    noticeEl.textContent = `Started job ${currentJob}.`;
    pollJob();
    pollTimer = setInterval(pollJob, 1000);
  } catch (err) {
    noticeEl.textContent = err.message;
    showStickyProgress("Update failed", err.message, 0, true);
    setButtons(false);
  }
}

async function pollJob() {
  if (!currentJob) return;
  try {
    const job = await api(`/api/update/status/${encodeURIComponent(currentJob)}`);
    const pct = job.total_bytes > 0 ? Math.floor((job.completed_bytes / job.total_bytes) * 100) : 0;
    const safePct = Math.max(0, Math.min(100, pct));
    const stage = statusLabel(job.status, job.message);
    barFillEl.style.width = `${safePct}%`;
    stickyBarFillEl.style.width = `${safePct}%`;
    percentEl.textContent = `${safePct}%`;
    stickyPercentEl.textContent = `${safePct}%`;
    currentAppEl.textContent = job.current_app || "-";
    stageEl.textContent = stage;
    fileCountEl.textContent = `${job.completed_files}/${job.total_files}`;
    byteCountEl.textContent = `${formatBytes(job.completed_bytes)} / ${formatBytes(job.total_bytes)}`;
    jobStateEl.textContent = job.message || "Waiting";
    stickyTitleEl.textContent = `${stage}: ${job.current_app || "-"}`;
    stickyDetailEl.textContent = `${job.completed_files}/${job.total_files} files | ${formatBytes(job.completed_bytes)} / ${formatBytes(job.total_bytes)} | ${job.message || ""}`;
    stickyProgressEl.classList.add("visible");
    stickyProgressEl.classList.toggle("failed", job.status === "failed");
    stickyProgressEl.classList.toggle("done", job.status === "success");
    jobLogEl.textContent = (job.app_results || []).map(r => `${r.application}: ${r.status}${r.error ? " - " + r.error : ""}`).join("\n");
    if (job.status === "success" || job.status === "failed") {
      clearInterval(pollTimer);
      setButtons(false);
      markCompletedApps(job.app_results || []);
      if (job.status === "success") {
        hideProgressTimer = setTimeout(hideStickyProgress, 1800);
      }
    }
  } catch (err) {
    jobStateEl.textContent = err.message;
    showStickyProgress("Progress error", err.message, 0, true);
  }
}

function selectedApps() {
  return [...selectedAppNames];
}

function updateSelectedSummary() {
  const selected = selectedApps();
  selectedCountEl.textContent = `${selected.length} selected`;
  if (!selected.length) {
    selectedListEl.innerHTML = "<li>No systems selected.</li>";
    return;
  }
  selectedListEl.innerHTML = selected.map(app => `
    <li>
      <span>${escapeHtml(app)}</span>
      <button class="remove-selected" type="button" data-remove-selected="${escapeHtml(app)}" title="Remove ${escapeHtml(app)}">×</button>
    </li>
  `).join("");
}

function markCompletedApps(results) {
  for (const result of results) {
    if (result.status !== "success") continue;
    completedApps[result.application] = true;
    const card = document.getElementById(`card-${safeId(result.application)}`);
    const mark = document.getElementById(`mark-${safeId(result.application)}`);
    if (card) card.classList.add("updated");
    if (mark) mark.textContent = "Done";
  }
}

function setButtons(disabled) {
  document.querySelectorAll("button").forEach(button => button.disabled = disabled);
}

function showStickyProgress(title, detail, percent, failed = false) {
  stickyProgressEl.classList.add("visible");
  stickyProgressEl.classList.toggle("failed", failed);
  stickyProgressEl.classList.remove("done");
  stickyTitleEl.textContent = title;
  stickyDetailEl.textContent = detail;
  stickyPercentEl.textContent = `${percent}%`;
  stickyBarFillEl.style.width = `${percent}%`;
}

function hideStickyProgress() {
  stickyProgressEl.classList.remove("visible");
}

function statusLabel(status, message) {
  const text = `${status || ""} ${message || ""}`.toLowerCase();
  if (text.includes("listing")) return "Checking files";
  if (text.includes("download")) return "Downloading";
  if (text.includes("backing")) return "Backing up";
  if (text.includes("install")) return "Installing";
  if (status === "success") return "Completed";
  if (status === "failed") return "Failed";
  return status || "Idle";
}

function formatBytes(value) {
  const n = Number(value || 0);
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

function safeId(value) {
  return String(value).replace(/[^A-Za-z0-9_-]/g, "_");
}

function escapeHtml(value) {
  return String(value).replace(/[&<>"']/g, ch => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch]));
}

function cssEscape(value) {
  if (window.CSS && CSS.escape) return CSS.escape(value);
  return String(value).replace(/["\\]/g, "\\$&");
}

document.getElementById("refresh").addEventListener("click", loadApps);
searchAppsEl.addEventListener("input", renderApps);
document.getElementById("updateSelected").addEventListener("click", () => startUpdate(selectedApps()));
document.getElementById("updateAll").addEventListener("click", () => startUpdate(apps.map(app => app.name)));
appsEl.addEventListener("click", event => {
  const checkApp = event.target.getAttribute("data-check");
  if (checkApp) upCheck(checkApp);
  const app = event.target.getAttribute("data-update");
  if (app) startUpdate([app]);
});
appsEl.addEventListener("change", event => {
  if (!event.target.matches("input[type=checkbox]")) return;
  if (event.target.checked) {
    selectedAppNames.add(event.target.value);
  } else {
    selectedAppNames.delete(event.target.value);
  }
  updateSelectedSummary();
});

selectedListEl.addEventListener("click", event => {
  const app = event.target.getAttribute("data-remove-selected");
  if (!app) return;
  selectedAppNames.delete(app);
  const checkbox = appsEl.querySelector(`input[type=checkbox][value="${cssEscape(app)}"]`);
  if (checkbox) checkbox.checked = false;
  updateSelectedSummary();
});

loadStatus().catch(err => machineEl.textContent = err.message);
loadApps();
