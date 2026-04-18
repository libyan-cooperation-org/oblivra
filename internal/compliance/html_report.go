package compliance

import (
	"bytes"
	"fmt"
	"html/template"
	"time"
)

// reportTemplate is the embedded HTML template for compliance reports.
const reportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }} — OBLIVRA Compliance Report</title>
  <style>
    :root {
      --bg: #0f1117;
      --surface: #1a1d27;
      --border: #2a2d3a;
      --text: #e4e6f0;
      --muted: #8b8fa3;
      --accent: #6366f1;
      --green: #22c55e;
      --red: #ef4444;
      --yellow: #eab308;
    }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: 'Inter', 'Segoe UI', system-ui, sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.6;
      padding: 2rem;
    }
    .container { max-width: 900px; margin: 0 auto; }
    .header {
      text-align: center;
      padding: 2rem;
      border-bottom: 2px solid var(--accent);
      margin-bottom: 2rem;
    }
    .header h1 { font-size: 1.8rem; color: var(--accent); }
    .header .subtitle { color: var(--muted); margin-top: 0.5rem; }
    .meta-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 1rem;
      margin: 1.5rem 0;
    }
    .meta-card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1rem;
    }
    .meta-card .label { font-size: 0.75rem; color: var(--muted); text-transform: uppercase; }
    .meta-card .value { font-size: 1.3rem; font-weight: 700; margin-top: 0.25rem; }
    .score-gauge {
      text-align: center;
      padding: 1.5rem;
      background: var(--surface);
      border-radius: 12px;
      border: 1px solid var(--border);
      margin: 1.5rem 0;
    }
    .score-value {
      font-size: 3rem;
      font-weight: 900;
    }
    .score-good { color: var(--green); }
    .score-warn { color: var(--yellow); }
    .score-bad { color: var(--red); }
    .section { margin: 2rem 0; }
    .section h2 {
      font-size: 1.2rem;
      border-bottom: 1px solid var(--border);
      padding-bottom: 0.5rem;
      margin-bottom: 1rem;
    }
    .finding {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1rem;
      margin: 0.75rem 0;
    }
    .finding-header { display: flex; justify-content: space-between; align-items: center; }
    .badge {
      padding: 2px 10px;
      border-radius: 4px;
      font-size: 0.75rem;
      font-weight: 700;
      text-transform: uppercase;
    }
    .badge-pass { background: rgba(34,197,94,0.15); color: var(--green); }
    .badge-fail { background: rgba(239,68,68,0.15); color: var(--red); }
    .badge-critical { background: rgba(239,68,68,0.25); color: var(--red); }
    .badge-warning { background: rgba(234,179,8,0.15); color: var(--yellow); }
    .badge-info { background: rgba(99,102,241,0.15); color: var(--accent); }
    .controls-table {
      width: 100%;
      border-collapse: collapse;
    }
    .controls-table th, .controls-table td {
      padding: 0.75rem;
      text-align: left;
      border-bottom: 1px solid var(--border);
    }
    .controls-table th {
      background: var(--surface);
      color: var(--muted);
      font-size: 0.75rem;
      text-transform: uppercase;
    }
    .footer {
      text-align: center;
      margin-top: 3rem;
      padding-top: 1rem;
      border-top: 1px solid var(--border);
      color: var(--muted);
      font-size: 0.8rem;
    }
    @media print {
      body { background: #fff; color: #111; }
      .meta-card, .finding, .score-gauge { border-color: #ccc; background: #f9f9f9; }
    }
  </style>
</head>
<body>
<div class="container">
  <div class="header">
    <h1>{{ .Title }}</h1>
    <div class="subtitle">{{ .Type }} Compliance Report • Generated {{ .GeneratedAt }}</div>
  </div>

  <div class="meta-grid">
    <div class="meta-card">
      <div class="label">Report Period</div>
      <div class="value">{{ .PeriodStart }} — {{ .PeriodEnd }}</div>
    </div>
    <div class="meta-card">
      <div class="label">Total Sessions</div>
      <div class="value">{{ .Summary.TotalSessions }}</div>
    </div>
    <div class="meta-card">
      <div class="label">Unique Hosts</div>
      <div class="value">{{ .Summary.UniquHosts }}</div>
    </div>
    <div class="meta-card">
      <div class="label">Total Commands</div>
      <div class="value">{{ .Summary.TotalCommands }}</div>
    </div>
  </div>

  <div class="score-gauge">
    <div class="label">Compliance Score</div>
    <div class="score-value {{ .ScoreClass }}">{{ printf "%.1f" .Summary.ComplianceScore }}%</div>
  </div>

  {{ if .Sections }}
  <div class="section">
    <h2>Assessment Sections</h2>
    <table class="controls-table">
      <thead>
        <tr>
          <th>Section</th>
          <th>Status</th>
          <th>Description</th>
        </tr>
      </thead>
      <tbody>
        {{ range .Sections }}
        <tr>
          <td><strong>{{ .Title }}</strong></td>
          <td><span class="badge {{ if eq .Status "pass" }}badge-pass{{ else }}badge-fail{{ end }}">{{ .Status }}</span></td>
          <td>{{ .Description }}</td>
        </tr>
        {{ end }}
      </tbody>
    </table>
  </div>
  {{ end }}

  {{ if .Findings }}
  <div class="section">
    <h2>Findings ({{ len .Findings }})</h2>
    {{ range .Findings }}
    <div class="finding">
      <div class="finding-header">
        <strong>{{ .Title }}</strong>
        <span class="badge {{ if eq .Severity "critical" }}badge-critical{{ else if eq .Severity "warning" }}badge-warning{{ else }}badge-info{{ end }}">{{ .Severity }}</span>
      </div>
      <p style="margin-top:0.5rem; color: var(--muted);">{{ .Description }}</p>
      {{ if .Remediation }}<p style="margin-top:0.25rem;"><em>Remediation:</em> {{ .Remediation }}</p>{{ end }}
    </div>
    {{ end }}
  </div>
  {{ end }}

  <div class="footer">
    OBLIVRA Security Platform • Report ID: {{ .ID }} • Generated {{ .GeneratedAt }}
  </div>
</div>
</body>
</html>`

// htmlReportData is the view model for the HTML template.
type htmlReportData struct {
	ID          string
	Title       string
	Type        string
	GeneratedAt string
	PeriodStart string
	PeriodEnd   string
	Summary     ReportSummary
	ScoreClass  string
	Sections    []ReportSection
	Findings    []Finding
}

// ExportHTML renders a compliance report as a branded HTML document.
func (g *ReportGenerator) ExportHTML(report *ComplianceReport) ([]byte, error) {
	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse report template: %w", err)
	}

	scoreClass := "score-good"
	if report.Summary.ComplianceScore < 50 {
		scoreClass = "score-bad"
	} else if report.Summary.ComplianceScore < 80 {
		scoreClass = "score-warn"
	}

	data := htmlReportData{
		ID:          report.ID,
		Title:       report.Title,
		Type:        string(report.Type),
		GeneratedAt: report.GeneratedAt,
		PeriodStart: report.PeriodStart,
		PeriodEnd:   report.PeriodEnd,
		Summary:     report.Summary,
		ScoreClass:  scoreClass,
		Sections:    report.Sections,
		Findings:    report.Findings,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute report template: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportPackResultHTML renders a compliance pack evaluation result as HTML.
func ExportPackResultHTML(result *PackResult) ([]byte, error) {
	const packTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{ .PackName }} — Compliance Evaluation</title>
  <style>
    :root { --bg: #0f1117; --surface: #1a1d27; --border: #2a2d3a; --text: #e4e6f0; --muted: #8b8fa3; --accent: #6366f1; --green: #22c55e; --red: #ef4444; }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: 'Inter', system-ui, sans-serif; background: var(--bg); color: var(--text); padding: 2rem; line-height: 1.6; }
    .container { max-width: 900px; margin: 0 auto; }
    h1 { color: var(--accent); margin-bottom: 0.5rem; }
    .score { font-size: 3rem; font-weight: 900; text-align: center; padding: 1rem; }
    .pass { color: var(--green); } .fail { color: var(--red); }
    table { width: 100%; border-collapse: collapse; margin: 1rem 0; }
    th, td { padding: 0.6rem; border-bottom: 1px solid var(--border); text-align: left; }
    th { background: var(--surface); color: var(--muted); font-size: 0.75rem; text-transform: uppercase; }
    .badge { padding: 2px 8px; border-radius: 4px; font-size: 0.7rem; font-weight: 700; }
    .badge-pass { background: rgba(34,197,94,0.15); color: var(--green); }
    .badge-fail { background: rgba(239,68,68,0.15); color: var(--red); }
  </style>
</head>
<body>
<div class="container">
  <h1>{{ .PackName }}</h1>
  <p style="color:var(--muted)">Evaluated: {{ .EvaluatedAt }} • {{ .PassedControls }}/{{ .TotalControls }} controls passed</p>
  <div class="score {{ if ge .Score 80.0 }}pass{{ else }}fail{{ end }}">{{ printf "%.1f" .Score }}%</div>
  <table>
    <thead><tr><th>Control</th><th>Title</th><th>Status</th><th>Checks</th></tr></thead>
    <tbody>
    {{ range .Controls }}
    <tr>
      <td>{{ .ID }}</td>
      <td>{{ .Title }}</td>
      <td><span class="badge {{ if .Passed }}badge-pass{{ else }}badge-fail{{ end }}">{{ if .Passed }}PASS{{ else }}FAIL{{ end }}</span></td>
      <td>{{ len .Checks }}</td>
    </tr>
    {{ end }}
    </tbody>
  </table>
</div>
</body>
</html>`

	tmpl, err := template.New("pack").Funcs(template.FuncMap{
		"ge": func(a, b float64) bool { return a >= b },
	}).Parse(packTemplate)
	if err != nil {
		return nil, err
	}

	viewData := struct {
		*PackResult
		EvaluatedAt string
	}{
		PackResult:  result,
		EvaluatedAt: result.EvaluatedAt,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, viewData); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ──────────────────────────────────────────────
// Extend ComplianceService with new methods
// ──────────────────────────────────────────────

// GeneratedAt field must exist on ComplianceReport for the template.
// This is ensured by adding it to the report struct generation in the GenerateReport method.
func init() {
	// Ensure the time package is available for report generation
	_ = time.Now()
}
