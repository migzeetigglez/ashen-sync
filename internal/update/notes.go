package update

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// NotesFile is written before relaunch so the new process can show What's New.
type NotesFile struct {
	Tag  string `json:"tag"`
	Body string `json:"body"`
}

// WriteNotesFile stores release notes in the OS temp dir and returns the path.
func WriteNotesFile(info Info) (string, error) {
	dir := filepath.Join(os.TempDir(), "save-channeler-update")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "whats-new.json")
	data, err := json.Marshal(NotesFile{Tag: info.Tag, Body: info.Body})
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// ReadNotesFile loads a What's New payload previously written by WriteNotesFile.
func ReadNotesFile(path string) (NotesFile, error) {
	var n NotesFile
	data, err := os.ReadFile(path)
	if err != nil {
		return n, err
	}
	err = json.Unmarshal(data, &n)
	return n, err
}
