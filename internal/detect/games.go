package detect

// Known Steam titles without native Steam Cloud for saves.
// Order matches the choose-game dialog / support list.
var Games = []Game{
	{
		ID:           "ds2-vanilla",
		Name:         "Dark Souls II",
		AppID:        "236430",
		FolderName:   "DarkSoulsII",
		ExeName:      "DarkSoulsII.exe",
		RemoteKey:    "dark-souls-ii",
		ProcessNames: []string{"DarkSoulsII", "DarkSoulsII.exe"},
	},
	{
		ID:           "ds2",
		Name:         "Dark Souls II: Scholar of the First Sin",
		AppID:        "335300",
		FolderName:   "DarkSoulsII",
		ExeName:      "DarkSoulsII.exe",
		RemoteKey:    "dark-souls-ii-sotfs",
		ProcessNames: []string{"DarkSoulsII", "DarkSoulsII.exe"},
	},
	{
		ID:           "ds3",
		Name:         "Dark Souls III",
		AppID:        "374320",
		FolderName:   "DarkSoulsIII",
		ExeName:      "DarkSoulsIII.exe",
		RemoteKey:    "dark-souls-iii",
		ProcessNames: []string{"DarkSoulsIII", "DarkSoulsIII.exe"},
	},
}

type Game struct {
	ID           string
	Name         string
	AppID        string
	FolderName   string // under AppData\Roaming
	ExeName      string // Windows executable filename in the install tree
	RemoteKey    string
	ProcessNames []string
}

func ByID(id string) (Game, bool) {
	switch id {
	case "ds2-sotfs", "sotfs":
		id = "ds2"
	case "darksouls3", "dsiii":
		id = "ds3"
	}
	for _, g := range Games {
		if g.ID == id {
			return g, true
		}
	}
	return Game{}, false
}

// SupportListName is the Arabic-numeral label used in support blurbs and the game dropdown.
func (g Game) SupportListName() string {
	switch g.ID {
	case "ds2-vanilla":
		return "Dark Souls 2"
	case "ds2":
		return "Dark Souls 2: Scholar of the First Sin"
	case "ds3":
		return "Dark Souls 3"
	default:
		return g.Name
	}
}

// SupportedNames returns support-list display names for every game Save Channeler syncs.
func SupportedNames() []string {
	out := make([]string, 0, len(Games))
	for _, g := range Games {
		out = append(out, g.SupportListName())
	}
	return out
}

// SelectLabels returns dropdown options (same strings as SupportedNames).
func SelectLabels() []string {
	return SupportedNames()
}

// SelectLabelForID returns the dropdown/support label for a game id (or alias).
func SelectLabelForID(id string) string {
	if g, ok := ByID(id); ok {
		return g.SupportListName()
	}
	return id
}

// IDFromSelectLabel maps a dropdown/support label (or raw/aliased id) to the canonical game id.
func IDFromSelectLabel(label string) string {
	for _, g := range Games {
		if g.SupportListName() == label {
			return g.ID
		}
	}
	if g, ok := ByID(label); ok {
		return g.ID
	}
	return label
}
