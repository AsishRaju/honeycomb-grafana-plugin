# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | ✅        |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

To report a security vulnerability, email **security@honeycomb.io** with:

- A description of the vulnerability and its potential impact
- Steps to reproduce or proof-of-concept code
- Any relevant logs or screenshots (sanitize any API keys first)

You will receive an acknowledgement within 48 hours and a resolution timeline within 5 business days.

## Security Design

This plugin is designed with the following security properties:

### API Key Handling
- The Honeycomb API key is stored in Grafana's `secureJsonData`, which is encrypted at rest by Grafana.
- The API key is **never** returned to the browser; all requests using the API key are made from the Go backend.
- The API key is never logged (even at DEBUG level).

### Network Requests
- All Honeycomb API calls are made from the Go backend plugin process, not from browser JavaScript.
- The plugin does not proxy arbitrary URLs — only the configured Honeycomb API endpoint is contacted.
- SSRF protection: the API URL is validated as a parseable URL on plugin initialization. Custom API URLs should be restricted to Honeycomb endpoints.

### Logs and Traces
- Query payloads are logged only at DEBUG level and do not include filter values or breakdown values that may contain PII.
- Rate limit headers are logged without any user-identifying information.

### Query Safety
- The plugin does not execute arbitrary backend code; it constructs Honeycomb API payloads from a structured query model.
- Raw JSON mode passes the user-provided JSON to the Honeycomb API as-is, without executing it locally.

### Frontend
- No secrets are stored in `jsonData` or panel state.
- Template variable values are substituted on the frontend before sending to the backend, but the substituted values do not include the API key.
