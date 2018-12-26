// +build gijit

package signal

// Package signal is not implemented for GOARCH=gijit.

func signal_disable(uint32) {}
func signal_enable(uint32)  {}
func signal_ignore(uint32)  {}
func signal_recv() uint32   { return 0 }

func loop() {}