package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// loadAuditPrompt creates user configuration only when an AUR audit needs it.
// Existing configuration is never rewritten, including during upgrades.
func loadAuditPrompt(path string) (string, error) {
	c, err := loadAuditConfigPath(path)
	if err != nil {
		return "", err
	}
	return c.prompt(), nil
}

func (c auditConfig) prompt() string {
	if c.Prompt != nil {
		return *c.Prompt
	}
	return auditPrompt()
}

func createAuditConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create audit configuration directory: %w", err)
	}
	prompt := auditPrompt()
	thinking := "medium"
	config, err := (auditConfig{Prompt: &prompt, Thinking: &thinking}).normalized()
	if err != nil {
		return err
	}
	defaults, marshalErr := json.MarshalIndent(config, "", "  ")
	if marshalErr != nil {
		return marshalErr
	}
	file, createErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if createErr == nil {
		_, writeErr := file.Write(append(defaults, '\n'))
		closeErr := file.Close()
		if writeErr != nil {
			return fmt.Errorf("write audit configuration: %w", writeErr)
		}
		if closeErr != nil {
			return closeErr
		}
	} else if !os.IsExist(createErr) {
		return fmt.Errorf("create audit configuration: %w", createErr)
	}
	return nil
}
