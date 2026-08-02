# Emacs / Eglot setup (community configuration)

Eglot supports one server per language, so use proxy mode. This is a community
configuration, protocol-validated but not an official integration.

```elisp
(with-eval-after-load 'eglot
  (add-to-list 'eglot-server-programs
               '(go-mode . ("batchweaver" "lsp" "--stdio"
                            "--proxy-gopls" "--gopls-command=gopls"))))
```

For sidecar use (BatchWeaver in addition to gopls), a multi-server setup such as
`eglot-booster` or running BatchWeaver as a separate flymake/LSP backend is
required; see your Eglot version's multi-server support.
