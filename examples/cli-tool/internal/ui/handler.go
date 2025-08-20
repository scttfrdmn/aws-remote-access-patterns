// Package ui provides rich user interface components for the CLI tool
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Handler provides rich UI functionality
type Handler struct {
	interactive bool
	useColor    bool
	reader      *bufio.Reader
}

// NewHandler creates a new UI handler
func NewHandler(interactive, useColor bool) *Handler {
	return &Handler{
		interactive: interactive,
		useColor:    useColor,
		reader:      bufio.NewReader(os.Stdin),
	}
}

// Colors for different output types
var (
	colorSuccess = color.New(color.FgGreen, color.Bold)
	colorError   = color.New(color.FgRed, color.Bold)
	colorWarning = color.New(color.FgYellow, color.Bold)
	colorInfo    = color.New(color.FgBlue, color.Bold)
	colorStep    = color.New(color.FgCyan, color.Bold)
	colorPrompt  = color.New(color.FgMagenta, color.Bold)
	colorBold    = color.New(color.Bold)
)

// SelectOption represents an option for selection prompts
type SelectOption struct {
	Value       string
	Label       string
	Description string
}

// ShowWelcome displays a welcome message with a banner
func (h *Handler) ShowWelcome(title, message string) {
	if !h.interactive {
		return
	}

	fmt.Println()
	h.printBanner(title)
	fmt.Println()
	fmt.Println(message)
	fmt.Println()
}

// ShowStep displays a step in a process
func (h *Handler) ShowStep(message string) {
	if h.useColor {
		colorStep.Printf("▶ %s\n", message)
	} else {
		fmt.Printf("▶ %s\n", message)
	}
}

// Success displays a success message
func (h *Handler) Success(message string) {
	if h.useColor {
		colorSuccess.Printf("✓ %s\n", message)
	} else {
		fmt.Printf("✓ %s\n", message)
	}
}

// Error displays an error message
func (h *Handler) Error(message string) {
	if h.useColor {
		colorError.Printf("✗ %s\n", message)
	} else {
		fmt.Printf("✗ %s\n", message)
	}
}

// Warning displays a warning message
func (h *Handler) Warning(message string) {
	if h.useColor {
		colorWarning.Printf("⚠ %s\n", message)
	} else {
		fmt.Printf("⚠ %s\n", message)
	}
}

// ShowInfo displays an informational message
func (h *Handler) ShowInfo(message string) {
	if h.useColor {
		colorInfo.Printf("ℹ %s\n", message)
	} else {
		fmt.Printf("ℹ %s\n", message)
	}
}

// ShowListItem displays a list item
func (h *Handler) ShowListItem(item string) {
	fmt.Printf("  • %s\n", item)
}

// Confirm prompts the user for yes/no confirmation
func (h *Handler) Confirm(message string) bool {
	if !h.interactive {
		return true // Default to yes in non-interactive mode
	}

	for {
		if h.useColor {
			colorPrompt.Printf("? %s [y/N]: ", message)
		} else {
			fmt.Printf("? %s [y/N]: ", message)
		}

		input, err := h.reader.ReadString('\n')
		if err != nil {
			return false
		}

		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Println("Please answer 'y' or 'n'")
		}
	}
}

// Prompt prompts the user for text input
func (h *Handler) Prompt(message, defaultValue string) (string, error) {
	if !h.interactive && defaultValue != "" {
		return defaultValue, nil
	}

	if defaultValue != "" {
		if h.useColor {
			colorPrompt.Printf("? %s [%s]: ", message, defaultValue)
		} else {
			fmt.Printf("? %s [%s]: ", message, defaultValue)
		}
	} else {
		if h.useColor {
			colorPrompt.Printf("? %s: ", message)
		} else {
			fmt.Printf("? %s: ", message)
		}
	}

	input, err := h.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "" && defaultValue != "" {
		return defaultValue, nil
	}

	return input, nil
}

// Select prompts the user to select from a list of options
func (h *Handler) Select(message string, options []SelectOption) (string, error) {
	if !h.interactive && len(options) > 0 {
		return options[0].Value, nil
	}

	fmt.Println()
	if h.useColor {
		colorPrompt.Printf("? %s\n\n", message)
	} else {
		fmt.Printf("? %s\n\n", message)
	}

	// Display options
	for i, option := range options {
		if h.useColor {
			fmt.Printf("  %s%d%s) %s\n", 
				colorBold.Sprint("["), 
				i+1,
				colorBold.Sprint("]"),
				option.Label)
		} else {
			fmt.Printf("  [%d] %s\n", i+1, option.Label)
		}
		
		if option.Description != "" {
			fmt.Printf("      %s\n", option.Description)
		}
		fmt.Println()
	}

	// Get user selection
	for {
		if h.useColor {
			colorPrompt.Printf("Selection [1]: ")
		} else {
			fmt.Printf("Selection [1]: ")
		}

		input, err := h.reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			input = "1"
		}

		selection, err := strconv.Atoi(input)
		if err != nil || selection < 1 || selection > len(options) {
			fmt.Printf("Please enter a number between 1 and %d\n", len(options))
			continue
		}

		return options[selection-1].Value, nil
	}
}

// ShowProgressBar displays a progress bar (simple implementation)
func (h *Handler) ShowProgressBar(current, total int, message string) {
	if !h.interactive {
		return
	}

	percentage := float64(current) / float64(total) * 100
	barWidth := 40
	filledWidth := int(float64(barWidth) * float64(current) / float64(total))

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("▒", barWidth-filledWidth)
	
	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)", message, bar, percentage, current, total)
	
	if current >= total {
		fmt.Println() // New line when complete
	}
}

// ShowSpinner displays a simple text-based spinner
func (h *Handler) ShowSpinner(message string, done chan bool) {
	if !h.interactive {
		fmt.Println(message)
		return
	}

	spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for {
		select {
		case <-done:
			fmt.Printf("\r%s ✓\n", message)
			return
		default:
			if h.useColor {
				fmt.Printf("\r%s %s", colorInfo.Sprint(spinChars[i]), message)
			} else {
				fmt.Printf("\r%s %s", spinChars[i], message)
			}
			time.Sleep(100 * time.Millisecond)
			i = (i + 1) % len(spinChars)
		}
	}
}

// ShowTable displays data in a table format
func (h *Handler) ShowTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println("No data to display")
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	h.printTableRow(headers, widths, true)
	h.printTableSeparator(widths)

	// Print rows
	for _, row := range rows {
		h.printTableRow(row, widths, false)
	}
}

// ShowUsageInstructions displays usage instructions after setup
func (h *Handler) ShowUsageInstructions() {
	fmt.Println()
	h.ShowInfo("Setup completed! Here are some commands to get started:")
	fmt.Println()
	fmt.Println("  datatool auth status              # Check authentication status")
	fmt.Println("  datatool s3 list                  # List S3 buckets")
	fmt.Println("  datatool ec2 instances            # List EC2 instances")
	fmt.Println("  datatool data sync --help         # Data synchronization options")
	fmt.Println("  datatool config show              # Show current configuration")
	fmt.Println()
	fmt.Println("For help with any command, use: datatool <command> --help")
	fmt.Println()
}

// printBanner prints an ASCII banner
func (h *Handler) printBanner(title string) {
	if h.useColor {
		colorBold.Printf("╔══════════════════════════════════════════════════════════════╗\n")
		colorBold.Printf("║  %-58s  ║\n", title)
		colorBold.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	} else {
		fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  %-58s  ║\n", title)
		fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	}
}

// printTableRow prints a table row
func (h *Handler) printTableRow(row []string, widths []int, isHeader bool) {
	fmt.Print("│")
	for i, cell := range row {
		if i < len(widths) {
			format := fmt.Sprintf(" %%-%ds │", widths[i])
			if isHeader && h.useColor {
				fmt.Print(colorBold.Sprintf(format, cell))
			} else {
				fmt.Printf(format, cell)
			}
		}
	}
	fmt.Println()
}

// printTableSeparator prints a table separator
func (h *Handler) printTableSeparator(widths []int) {
	fmt.Print("├")
	for i, width := range widths {
		fmt.Print(strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			fmt.Print("┼")
		}
	}
	fmt.Println("┤")
}