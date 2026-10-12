package cmd

import "testing"

func TestPoolCommandRegistered(t *testing.T) {
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "pool" {
			return
		}
	}
	t.Fatal("expected pool command registered on root")
}
