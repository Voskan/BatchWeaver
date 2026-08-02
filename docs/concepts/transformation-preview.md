# Transformation preview

BatchWeaver never rewrites your source implicitly. A transformation is first
previewed: the editor opens a read-only view showing the operation binding,
structural context, candidate evidence, and the exact CLI commands
(`batchweaver prove`, `batchweaver transform diff`) that produce the deterministic
diff and proof certificate.

Applying a transformation is a separate, explicit action gated on a current
proof, unchanged documents, a type-checking package, and client support for
versioned workspace edits. Edits that require generated files use the Prompt 06
materialization command, only after explicit consent, with backup and revert.
