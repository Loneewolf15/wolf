package wolf

import "embed"

//go:embed runtime/*.c runtime/*.h third_party/lib/*/* third_party/include/*
var Assets embed.FS
