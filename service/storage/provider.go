package storage

import "context"

// Provider is the interface for storage providers
type Provider interface {
	// Read reads the object from storage
	Read(ctx context.Context, path string) ([]byte, error)
	// Write writes the object to storage
	Write(ctx context.Context, path string, data []byte)
}
