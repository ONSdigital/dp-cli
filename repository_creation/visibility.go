package repository_creation

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

const CreationCancelledMessage = "Repository creation cancelled. No repository was created."

// ErrCreationCancelled indicates that the user declined public repository creation.
var ErrCreationCancelled = errors.New("public repository creation cancelled")

func selectVisibility(visibility string, input io.Reader, output io.Writer) (string, error) {
	scanner := bufio.NewScanner(input)
	readInput := func() (string, error) {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("reading repository visibility input: %w", err)
			}
			return "", fmt.Errorf("reading repository visibility input: %w", io.EOF)
		}
		return strings.ToLower(strings.TrimSpace(scanner.Text())), nil
	}

	if visibility == "" {
		for {
			fmt.Fprintln(output, "Please select repository visibility (Enter defaults to internal):")
			fmt.Fprintln(output, "[0] internal (default)\n[1] private\n[2] public (WARNING: visible to everyone outside ONSdigital)")
			response, err := readInput()
			if err != nil {
				return "", err
			}
			switch response {
			case "", "0", "internal":
				visibility = "internal"
			case "1", "private":
				visibility = "private"
			case "2", "public":
				visibility = "public"
			default:
				fmt.Fprintln(output, "Invalid visibility; choose internal, private or public.")
				continue
			}
			break
		}
	}

	switch visibility {
	case "internal", "private":
		return visibility, nil
	case "public":
		fmt.Fprintln(output, "WARNING: A public repository and its contents will be visible to everyone outside ONSdigital.")
		for {
			fmt.Fprintln(output, "Create a public repository? [y/N]:")
			response, err := readInput()
			if err != nil {
				return "", err
			}
			switch response {
			case "y", "yes":
				return visibility, nil
			case "", "n", "no":
				return "", ErrCreationCancelled
			default:
				fmt.Fprintln(output, "Please answer yes or no.")
			}
		}
	default:
		return "", fmt.Errorf("invalid repository visibility %q: must be internal, private or public", visibility)
	}
}
