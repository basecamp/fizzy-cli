// Package skills embeds the skill files in the binary.
package skills

import "embed"

//go:embed fizzy
var FS embed.FS
