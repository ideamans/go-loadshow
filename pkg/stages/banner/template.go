package banner

import (
	"bytes"
	"fmt"
	"html/template"
	"time"
)

// TemplateVars contains variables for the banner HTML template.
type TemplateVars struct {
	BodyWidth       int
	MainTitle       string
	SubTitle        string
	Credit          string
	CreatedAt       string
	TrafficLabel    string
	TrafficValue    string
	OnLoadTimeLabel string
	OnLoadTimeValue string
}

// NewTemplateVars creates template variables from banner input.
func NewTemplateVars(width int, url, title string, loadTimeMs int, totalBytes int64, credit string) TemplateVars {
	if credit == "" {
		credit = "loadshow"
	}
	return TemplateVars{
		BodyWidth:       width,
		MainTitle:       title,
		SubTitle:        url,
		Credit:          credit,
		CreatedAt:       time.Now().Format("2006/01/02 15:04:05"),
		TrafficLabel:    "Traffic",
		TrafficValue:    fmt.Sprintf("%.2f MB", float64(totalBytes)/1024/1024),
		OnLoadTimeLabel: "OnLoad Time",
		OnLoadTimeValue: fmt.Sprintf("%.2f sec.", float64(loadTimeMs)/1000),
	}
}

// RenderHTML renders the banner HTML template with the given variables.
func RenderHTML(vars TemplateVars) (string, error) {
	tmpl, err := template.New("banner").Parse(defaultHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// defaultHTMLTemplate is the banner HTML template.
const defaultHTMLTemplate = `<html>
  <head>
    <style>
      * {
        margin: 0;
        padding: 0;
        box-sizing: border-box;
        white-space: nowrap;
      }
      html, body {
        height: auto;
      }
      body {
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        width: {{.BodyWidth}}px;
        padding: 10px 12px;
        background-color: #f5f5f5;
        display: inline-flex;
        flex-direction: column;
        gap: 6px;
      }
      .ellipsis {
        overflow: hidden;
        text-overflow: ellipsis;
      }
      .header {
        display: flex;
        flex-direction: column;
        gap: 2px;
      }
      .main-title {
        font-size: 15px;
        font-weight: 600;
        color: #222;
      }
      .sub-title {
        font-size: 11px;
        color: #0066cc;
      }
      .meta {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 6px 0;
        border-bottom: 1px solid #ddd;
      }
      .credit {
        font-size: 14px;
        font-weight: 500;
        color: #333;
      }
      .datetime {
        font-size: 12px;
        color: #666;
      }
      .properties {
        display: flex;
        align-items: center;
        gap: 0;
        padding-top: 6px;
      }
      .prop {
        display: flex;
        align-items: center;
        gap: 6px;
      }
      .prop-label {
        font-size: 12px;
        color: #666;
      }
      .prop-value {
        font-size: 14px;
        font-weight: 600;
        color: #333;
      }
      .prop-divider {
        width: 1px;
        height: 16px;
        background-color: #ccc;
        margin: 0 12px;
      }
    </style>
  </head>
  <body>
    <div class="header">
      <div class="main-title ellipsis">{{.MainTitle}}</div>
      <div class="sub-title ellipsis">{{.SubTitle}}</div>
    </div>
    <div class="meta">
      <div class="credit">{{.Credit}}</div>
      <div class="datetime">{{.CreatedAt}}</div>
    </div>
    <div class="properties">
      <div class="prop">
        <span class="prop-label">{{.TrafficLabel}}</span>
        <span class="prop-value">{{.TrafficValue}}</span>
      </div>
      <div class="prop-divider"></div>
      <div class="prop">
        <span class="prop-label">{{.OnLoadTimeLabel}}</span>
        <span class="prop-value">{{.OnLoadTimeValue}}</span>
      </div>
    </div>
  </body>
</html>`
