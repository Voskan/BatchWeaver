# Launch Decision: v0.1.0-beta.2

Decision: **authorized hotfix beta**.

Beta.2 is required because public `go install` verification found a P1 version
metadata defect in immutable beta.1. The new version preserves semantic-version
immutability, changes no public schema or ABI, and adds focused regression
coverage. Stable v1 remains blocked by the evidence, migration, API-freeze, and
governance criteria in the stable-release decision.
