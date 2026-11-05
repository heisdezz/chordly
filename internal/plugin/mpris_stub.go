//go:build !linux

package plugin

import "errors"

// MPRISServer is a no-op stub on non-Linux operating systems.
type MPRISServer struct{}

// NewMPRISServer returns an unsupported error on non-Linux platforms.
func NewMPRISServer() (*MPRISServer, error) {
	return nil, errors.New("mpris is only supported on linux")
}

func (s *MPRISServer) BusName() string             { return "" }
func (s *MPRISServer) SetQuitHandler(fn func())    {}
func (s *MPRISServer) SetRaiseHandler(fn func())   {}
func (s *MPRISServer) Close() error                { return nil }
