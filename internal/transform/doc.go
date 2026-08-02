// Package transform implements BatchWeaver's first end-to-end transformation
// path: consuming semantic proof certificates, planning deterministic
// source-preserving rewrites, generating the first production transformation
// (static slice/array loop prefetch for certified read-only operations), and
// executing transformed code through Go build overlays without modifying the
// source tree.
//
// The package never transforms a candidate unless a current, valid,
// strategy-specific proof certificate proves it eligible. It produces a
// versioned transformation IR that references stable analysis identities and
// proof certificates and never serializes AST nodes, SSA values, token
// positions, or pointer identities.
//
// Transformation is non-mutating by default: plans are applied through a
// standard Go `-overlay` file so build, test, and run observe the transformed
// bytes while the working tree is untouched. Materialization is a separate,
// explicit, atomic, reversible operation with a backup manifest.
package transform
