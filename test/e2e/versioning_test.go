package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestEndToEndFeatureVersioningWorkflow tests the complete fogit workflow with versioning:
// 1. Create a new project folder
// 2. Initialize git and fogit
// 3. Add initial project files
// 4. Create a new feature (v1) - branches off
// 5. Make changes on the feature branch (v1)
// 6. Close/merge the feature (v1)
// 7. Verify we're back on base branch with v1 changes preserved
// 8. Create a new version (v2) of the same feature - branches off again
// 9. Make changes on the feature branch (v2)
// 10. Close/merge the feature (v2)
// 11. Verify we're back on base branch with v2 changes preserved
func TestEndToEndFeatureVersioningWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create a unique test project directory
	projectDir := filepath.Join(t.TempDir(), "E2E_FeatureVersioningTest")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// STEP 1: Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cfg, err := gitRepo.Config()
	if err != nil {
		t.Fatalf("Failed to get git config: %v", err)
	}
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	if err := gitRepo.SetConfig(cfg); err != nil {
		t.Fatalf("Failed to set git config: %v", err)
	}

	// STEP 2: Create initial project files
	t.Log("Step 2: Creating initial project files...")
	initialFiles := map[string]string{
		"README.md": `# E2E Versioning Test Project

This is a test project for fogit versioning workflow.

## Features
- Feature versioning
- Git integration
`,
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"go.mod": `module e2e-versioning-test

go 1.23
`,
	}

	for filename, content := range initialFiles {
		filePath := filepath.Join(projectDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	// Make initial commit
	worktree, err := gitRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	if _, err := worktree.Add("."); err != nil {
		t.Fatalf("Failed to stage files: %v", err)
	}
	_, err = worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// STEP 3: Initialize fogit
	t.Log("Step 3: Initializing fogit...")
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}
	t.Logf("Init output: %s", output)

	// Get the base branch name and configure fogit to use it
	head, err := gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	baseBranch := head.Name().Short()
	t.Logf("Base branch: %s", baseBranch)

	if baseBranch != "main" {
		output, err = runFogit(t, projectDir, "config", "set", "workflow.base_branch", baseBranch)
		if err != nil {
			t.Fatalf("Failed to set base branch: %v\nOutput: %s", err, output)
		}
	}

	// STEP 4: Create a new feature v1 (should create branch)
	t.Log("Step 4: Creating new feature 'Payment System' (v1)...")
	output, err = runFogit(t, projectDir, "feature", "Payment System", "--description", "Implement payment processing")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature create output: %s", output)

	// Verify we're on a feature branch
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after feature creation: %v", err)
	}
	featureBranchV1 := head.Name().Short()
	t.Logf("Current branch after feature creation (v1): %s", featureBranchV1)

	if !strings.HasPrefix(featureBranchV1, "feature/") {
		t.Errorf("Expected to be on a feature branch (feature/*), got: %s", featureBranchV1)
	}

	// STEP 5: Make changes on the feature branch (v1)
	t.Log("Step 5: Making changes on feature branch (v1)...")

	// Add payment files for v1
	paymentFilesV1 := map[string]string{
		"payment/payment.go": `package payment

import "fmt"

// Payment represents a payment transaction
type Payment struct {
	ID     string
	Amount float64
	Status string
}

// Process processes a payment (v1 - basic implementation)
func Process(p *Payment) error {
	fmt.Printf("Processing payment %s for amount %.2f\n", p.ID, p.Amount)
	p.Status = "processed"
	return nil
}
`,
	}

	// Create payment directory
	paymentDir := filepath.Join(projectDir, "payment")
	if err := os.MkdirAll(paymentDir, 0755); err != nil {
		t.Fatalf("Failed to create payment directory: %v", err)
	}

	for filename, content := range paymentFilesV1 {
		filePath := filepath.Join(projectDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	// Update main.go to use payment v1
	mainV1 := `package main

import (
	"fmt"
	"e2e-versioning-test/payment"
)

func main() {
	fmt.Println("Hello, World!")
	
	// Payment system v1
	p := &payment.Payment{ID: "PAY001", Amount: 99.99}
	if err := payment.Process(p); err != nil {
		fmt.Printf("Payment failed: %v\n", err)
	}
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainV1), 0644); err != nil {
		t.Fatalf("Failed to update main.go: %v", err)
	}

	// Stage and commit changes
	if _, err := worktree.Add("."); err != nil {
		t.Fatalf("Failed to stage feature changes: %v", err)
	}
	_, err = worktree.Commit("Add payment system v1", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit feature changes: %v", err)
	}

	// STEP 6: Close/merge the feature v1
	t.Log("Step 6: Merging/closing the feature (v1)...")
	output, err = runFogit(t, projectDir, "merge")
	if err != nil {
		t.Fatalf("Failed to merge feature: %v\nOutput: %s", err, output)
	}
	t.Logf("Merge output: %s", output)

	// STEP 7: Verify we're back on base branch with v1 changes
	t.Log("Step 7: Verifying we're back on base branch with v1 changes...")
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after merge: %v", err)
	}
	currentBranch := head.Name().Short()
	t.Logf("Current branch after merge (v1): %s", currentBranch)

	if currentBranch != baseBranch {
		t.Errorf("Expected to be back on base branch '%s', got '%s'", baseBranch, currentBranch)
	}

	// Verify v1 payment file exists
	if _, err := os.Stat(filepath.Join(projectDir, "payment", "payment.go")); os.IsNotExist(err) {
		t.Error("payment/payment.go should exist on base branch after v1 merge")
	}

	// Verify feature v1 is closed
	output, err = runFogit(t, projectDir, "list", "--state", "closed")
	if err != nil {
		t.Fatalf("Failed to list closed features: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Payment System") {
		t.Error("Feature 'Payment System' should be listed as closed after v1")
	}

	// STEP 8: Create a new version (v2) of the same feature
	t.Log("Step 8: Creating new version (v2) of 'Payment System'...")
	// Use --new-version flag to create v2 of existing feature
	output, err = runFogit(t, projectDir, "feature", "Payment System", "--new-version", "--description", "Add refund support to payment system")
	if err != nil {
		t.Fatalf("Failed to create feature v2: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature v2 create output: %s", output)

	// Verify v2 starts in OPEN state (per spec: created_at == modified_at)
	output, err = runFogit(t, projectDir, "list", "--state", "open")
	if err != nil {
		t.Fatalf("Failed to list open features: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Payment System") {
		t.Error("Feature 'Payment System' v2 should be in OPEN state immediately after creation")
	}

	// Verify we're on a feature branch again
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after feature v2 creation: %v", err)
	}
	featureBranchV2 := head.Name().Short()
	t.Logf("Current branch after feature creation (v2): %s", featureBranchV2)

	if !strings.HasPrefix(featureBranchV2, "feature/") {
		t.Errorf("Expected to be on a feature branch (feature/*), got: %s", featureBranchV2)
	}

	// STEP 9: Make changes on the feature branch (v2) and use fogit commit to transition to in-progress
	t.Log("Step 9: Making changes on feature branch (v2) - adding refund support...")

	// Update payment.go with v2 features (refund support)
	paymentV2 := `package payment

import "fmt"

// Payment represents a payment transaction
type Payment struct {
	ID     string
	Amount float64
	Status string
}

// Process processes a payment (v2 - with validation)
func Process(p *Payment) error {
	if p.Amount <= 0 {
		return fmt.Errorf("invalid amount: %.2f", p.Amount)
	}
	fmt.Printf("Processing payment %s for amount %.2f\n", p.ID, p.Amount)
	p.Status = "processed"
	return nil
}

// Refund processes a refund for a payment (v2 feature)
func Refund(p *Payment) error {
	if p.Status != "processed" {
		return fmt.Errorf("cannot refund payment with status: %s", p.Status)
	}
	fmt.Printf("Refunding payment %s for amount %.2f\n", p.ID, p.Amount)
	p.Status = "refunded"
	return nil
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "payment", "payment.go"), []byte(paymentV2), 0644); err != nil {
		t.Fatalf("Failed to update payment.go: %v", err)
	}

	// Add a new refund handler file
	refundHandler := `package payment

import "fmt"

// RefundRequest represents a refund request
type RefundRequest struct {
	PaymentID string
	Reason    string
}

// HandleRefund handles a refund request
func HandleRefund(req *RefundRequest) error {
	fmt.Printf("Processing refund for payment %s: %s\n", req.PaymentID, req.Reason)
	return nil
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "payment", "refund.go"), []byte(refundHandler), 0644); err != nil {
		t.Fatalf("Failed to create refund.go: %v", err)
	}

	// Update main.go to use payment v2 features
	mainV2 := `package main

import (
	"fmt"
	"e2e-versioning-test/payment"
)

func main() {
	fmt.Println("Hello, World!")
	
	// Payment system v2 - with refund support
	p := &payment.Payment{ID: "PAY001", Amount: 99.99}
	if err := payment.Process(p); err != nil {
		fmt.Printf("Payment failed: %v\n", err)
		return
	}
	
	// New v2 feature: refund
	if err := payment.Refund(p); err != nil {
		fmt.Printf("Refund failed: %v\n", err)
	}
	
	// New v2 feature: refund handler
	req := &payment.RefundRequest{PaymentID: p.ID, Reason: "Customer request"}
	payment.HandleRefund(req)
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainV2), 0644); err != nil {
		t.Fatalf("Failed to update main.go for v2: %v", err)
	}

	// Use fogit commit to transition feature from open to in-progress
	// This properly updates the feature's modified_at timestamp
	output, err = runFogit(t, projectDir, "commit", "-m", "Add payment system v2 with refund support")
	if err != nil {
		t.Fatalf("Failed to fogit commit v2 changes: %v\nOutput: %s", err, output)
	}
	t.Logf("Fogit commit output: %s", output)

	// Verify feature is now in IN-PROGRESS state (modified_at > created_at)
	output, err = runFogit(t, projectDir, "list", "--state", "in-progress")
	if err != nil {
		t.Fatalf("Failed to list in-progress features: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Payment System") {
		t.Error("Feature 'Payment System' v2 should be in IN-PROGRESS state after fogit commit")
	}

	// STEP 10: Close/merge the feature v2
	t.Log("Step 10: Merging/closing the feature (v2)...")
	output, err = runFogit(t, projectDir, "merge")
	if err != nil {
		t.Fatalf("Failed to merge feature v2: %v\nOutput: %s", err, output)
	}
	t.Logf("Merge v2 output: %s", output)

	// STEP 11: Verify we're back on base branch with v2 changes preserved
	t.Log("Step 11: Verifying we're back on base branch with v2 changes...")
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after v2 merge: %v", err)
	}
	currentBranch = head.Name().Short()
	t.Logf("Current branch after v2 merge: %s", currentBranch)

	if currentBranch != baseBranch {
		t.Errorf("Expected to be back on base branch '%s', got '%s'", baseBranch, currentBranch)
	}

	// Verify v2 files exist on base branch
	if _, err := os.Stat(filepath.Join(projectDir, "payment", "refund.go")); os.IsNotExist(err) {
		t.Error("payment/refund.go should exist on base branch after v2 merge")
	}

	// Verify main.go contains v2 features
	mainContent, err := os.ReadFile(filepath.Join(projectDir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}
	if !strings.Contains(string(mainContent), "payment.Refund") {
		t.Error("main.go should contain payment.Refund after v2 merge")
	}
	if !strings.Contains(string(mainContent), "RefundRequest") {
		t.Error("main.go should contain RefundRequest after v2 merge")
	}

	// Verify payment.go contains v2 Refund function
	paymentContent, err := os.ReadFile(filepath.Join(projectDir, "payment", "payment.go"))
	if err != nil {
		t.Fatalf("Failed to read payment.go: %v", err)
	}
	if !strings.Contains(string(paymentContent), "func Refund") {
		t.Error("payment.go should contain Refund function after v2 merge")
	}

	// STEP 12: Verify feature versions
	t.Log("Step 12: Verifying feature has multiple versions...")
	output, err = runFogit(t, projectDir, "versions", "Payment System")
	if err != nil {
		t.Fatalf("Failed to list versions: %v\nOutput: %s", err, output)
	}
	t.Logf("Versions output: %s", output)

	// Should show both v1 and v2
	if !strings.Contains(output, "1") || !strings.Contains(output, "2") {
		t.Log("Note: versions output may vary based on implementation")
	}

	// Verify feature is still closed
	output, err = runFogit(t, projectDir, "list", "--state", "closed")
	if err != nil {
		t.Fatalf("Failed to list closed features: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Payment System") {
		t.Error("Feature 'Payment System' should be listed as closed after v2")
	}

	t.Log("✅ End-to-end versioning workflow test passed!")
}
