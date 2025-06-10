package protolite

import (
	"errors"
	"github.com/protolite/registry"
)

// Protolite is the main interface for the library.
type Protolite interface {
	// Parse parses the given data into a map of string to interface. This is used when schema is not known.
	Parse(data []byte) (map[string]interface{}, error)
	Marshal(data map[string]interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type protolite struct {
	registry *registry.Registry
}

// Marshal implements Protolite.
func (p *protolite) Marshal(data map[string]interface{}) ([]byte, error) {
	//TODO: Implement
	return nil, errors.New("not implemented")
}

// Parse implements Protolite.
func (p *protolite) Parse(data []byte) (map[string]interface{}, error) {
	//TODO: Implement
	return nil, errors.New("not implemented")
}

// Unmarshal implements Protolite.
func (p *protolite) Unmarshal(data []byte, v interface{}) error {
	//TODO: Implement
	return errors.New("not implemented")
}

func NewProtolite() Protolite {
	return &protolite{
		registry: registry.NewRegistry(),
	}
}
