package detect

// Known Steam titles without native Steam Cloud for saves.
var Games = []Game{
	{
		ID:           "ds2",
		Name:         "DARK SOULS II: Scholar of the First Sin",
		AppID:        "335300",
		FolderName:   "DarkSoulsII",
		RemoteKey:    "dark-souls-ii-sotfs",
		ProcessNames: []string{"DarkSoulsII", "DarkSoulsII.exe"},
	},
	{
		ID:           "ds2-vanilla",
		Name:         "DARK SOULS II",
		AppID:        "236430",
		FolderName:   "DarkSoulsII",
		RemoteKey:    "dark-souls-ii",
		ProcessNames: []string{"DarkSoulsII", "DarkSoulsII.exe"},
	},
	{
		ID:           "ds3",
		Name:         "DARK SOULS III",
		AppID:        "374320",
		FolderName:   "DarkSoulsIII",
		RemoteKey:    "dark-souls-iii",
		ProcessNames: []string{"DarkSoulsIII", "DarkSoulsIII.exe"},
	},
}

type Game struct {
	ID           string
	Name         string
	AppID        string
	FolderName   string // under AppData\Roaming
	RemoteKey    string
	ProcessNames []string
}

func ByID(id string) (Game, bool) {
	for _, g := range Games {
		if g.ID == id {
			return g, true
		}
	}
	// aliases
	switch id {
	case "ds2-sotfs", "sotfs":
		return Games[0], true
	case "darksouls3", "dsiii":
		return Games[2], true
	}
	return Game{}, false
}
