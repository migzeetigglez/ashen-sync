package version

// Version is the Save Channeler release version (no leading "v").
// Packaged builds override this via:
//
//	-ldflags "-X github.com/migzeetigglez/ashen-sync/internal/version.Version=0.2.0"
var Version = "0.2.0"
