// SPDX-License-Identifier: Apache-2.0
package archdoc

import (
	"strings"
	"testing"
)

// requiredMentions guards significant exported symbols that the
// package-completeness check alone cannot catch.
var requiredMentions = []string{
	"VisionPool", "LlamaCppDeployer", "VisionSlot", "OllamaProvider",
	"KimiProvider", "AsticaProvider", "FallbackProvider",
	"NavigationGraph", "Analyzer",
}

func TestArchitectureDocAccuracy(t *testing.T) {
	root, err := ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	problems, err := Verify(root, requiredMentions)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("docs/ARCHITECTURE.md is inaccurate (%d problems):\n  - %s",
			len(problems), strings.Join(problems, "\n  - "))
	}
}
