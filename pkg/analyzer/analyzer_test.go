package analyzer

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

func TestAnalyzeScholarships(t *testing.T) {
	// Create a pipe to capture the output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	// Keep the original stdout
	stdout := os.Stdout
	// Redirect stdout to the pipe
	os.Stdout = w

	scholarships, err := ReadScholarships("../../output.csv")
	if err != nil {
		t.Fatalf("Failed to read scholarships: %v", err)
	}

	AnalyzeScholarships(scholarships)

	// Close the writer
	w.Close()
	// Restore the original stdout
	os.Stdout = stdout

	// Read the output from the pipe
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		log.Fatalf("Failed to read from pipe: %v", err)
	}

	output := buf.String()

	// Verify the output
	expectedLines := []string{
		"Total de bolsas: 57",
		"- 2023: 37",
		"- 2022: 20",
		"- Unidade A: 18",
		"- Unidade B: 12",
		"- Unidade C: 27",
		"Unidade com mais bolsas: Unidade C (27 bolsas)",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Expected output to contain:\n%s\n\nGot:\n%s", line, output)
		}
	}
}
