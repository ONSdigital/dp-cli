package eks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Prompts the user with an explanation and caches sudo credentials using -v (validate flag).
// This should be called once at the start of operations that require elevated privileges.
// Subsequent sudo calls within the timeout window (typically 5 minutes) won't prompt again.
func EnsureSudo(reason string) error {
	// Check if sudo is already cached (no prompt needed)
	if exec.CommandContext(context.Background(), "sudo", "-n", "true").Run() == nil {
		return nil
	}

	// Explain why we need sudo
	fmt.Println()
	fmt.Printf("  sudo access required: %s\n", reason)
	fmt.Println("  Enter your password when prompted.")
	fmt.Println()

	// Run sudo -v to cache credentials (this will prompt)
	cmd := exec.CommandContext(context.Background(), "sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to obtain sudo access: %w", err)
	}

	return nil
}
