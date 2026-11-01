// Package groundwork embeds the skill's machine-readable index, so the groundwork command can check a
// run log in any project the skill is used in, not only in this repo.
package groundwork

import _ "embed"

// Index is skills/groundwork/index.json as built into the command.
//
//go:embed skills/groundwork/index.json
var Index []byte
