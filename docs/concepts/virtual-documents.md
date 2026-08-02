# Virtual documents

Some BatchWeaver views have no physical file: a proof certificate, a
transformation diff, an operation graph, or a tuning report. These are surfaced as
read-only virtual documents (and, in the VS Code extension, as opened text
documents) bound to the snapshot that produced them.

Virtual content is deterministic, size-limited, and privacy-safe: it contains no
secret values and never grants access to arbitrary paths. The always-available
text/DOT/JSON forms mean every view is usable without a browser or webview.
