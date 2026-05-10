package wolf

import "embed"

//go:embed runtime/*.c runtime/*.h third_party/lib/*/*
var Assets embed.FS
