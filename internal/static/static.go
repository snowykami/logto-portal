package static

import "embed"

//go:embed all:dist
var dist embed.FS

func FS() embed.FS {
	return dist
}
