package main

import (
	"fmt"
)

// RepoNotFoundError represents the error for missing repo.
type RepoNotFoundError struct {
	repo string
}

// See errors.Error()
func (e RepoNotFoundError) Error() string {
	if e.repo == "" {
		return fmt.Sprintf("no repository given")
	}
	return fmt.Sprintf("repository '%v' not found", e.repo)
}

// Error represents a generic error.
type Error struct {
	repo string
	msg  string
}

// See errors.Error()
func (e Error) Error() string {
	return fmt.Sprintf("repository '%v': %v", e.repo, e.msg)
}
