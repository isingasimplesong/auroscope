package app

import (
	"encoding/json"
	"fmt"
	"os"
)

type recipeDrift struct {
	Pkgbase string `json:"pkgbase"`
}

func (d recipeDrift) Error() string {
	return fmt.Sprintf("the recipe for %s changed after approval (recipe identity drift); build stopped before executing the changed recipe", escapeTerminal(d.Pkgbase))
}

func runGuard(txPath, worktree string) error {
	if worktree == "" {
		var err error
		worktree, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	pkgbase := os.Getenv("PKGBASE")
	if pkgbase == "" {
		return fmt.Errorf("PKGBASE is not set")
	}
	data, err := os.ReadFile(txPath)
	if err != nil {
		return fmt.Errorf("read transaction file: %w", err)
	}
	var approved []recipeIdentity
	if err := json.Unmarshal(data, &approved); err != nil {
		return fmt.Errorf("parse transaction file: %w", err)
	}
	var expected *recipeIdentity
	for i := range approved {
		if approved[i].Pkgbase == pkgbase {
			expected = &approved[i]
			break
		}
	}
	if expected == nil {
		return fmt.Errorf("unexpected pkgbase %q", pkgbase)
	}
	actual, _, err := readRecipeIdentity(pkgbase, worktree)
	if err != nil {
		return err
	}
	if actual.Commit != expected.Commit || actual.ManifestDigest != expected.ManifestDigest {
		drift := recipeDrift{Pkgbase: pkgbase}
		data, err := json.Marshal(drift)
		if err != nil {
			return err
		}
		// This private, attempt-local result is independent of Paru's localized
		// diagnostics and exit status. It never authorizes a recipe.
		if err := os.WriteFile(txPath+".drift.json", data, 0o600); err != nil {
			return fmt.Errorf("%s; cannot report drift to AURoscope: %w", drift, err)
		}
		return drift
	}
	return nil
}
