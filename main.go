package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// NessusClientData represents the root element of a Nessus XML file
type NessusClientData struct {
	XMLName xml.Name `xml:"NessusClientData_v2"`
	Policy  Policy   `xml:"Policy"`
	Report  Report   `xml:"Report"`
}

// Policy represents the policy section
type Policy struct {
	XMLName xml.Name `xml:"Policy"`
	Content string   `xml:",innerxml"`
}

// Report represents the report section
type Report struct {
	XMLName     xml.Name     `xml:"Report"`
	Name        string       `xml:"name,attr"`
	ReportHosts []ReportHost `xml:"ReportHost"`
}

// ReportHost represents a single host
type ReportHost struct {
	XMLName        xml.Name       `xml:"ReportHost"`
	Name           string         `xml:"name,attr"`
	HostProperties HostProperties `xml:"HostProperties"`
	ReportItems    []ReportItem   `xml:"ReportItem"`
}

// HostProperties represents host properties
type HostProperties struct {
	XMLName xml.Name `xml:"HostProperties"`
	Content string   `xml:",innerxml"`
}

// ReportItem represents a single finding/vulnerability
type ReportItem struct {
	XMLName      xml.Name `xml:"ReportItem"`
	Content      string   `xml:",innerxml"`
	Port         string   `xml:"port,attr"`
	SvcName      string   `xml:"svc_name,attr"`
	Protocol     string   `xml:"protocol,attr"`
	Severity     string   `xml:"severity,attr"`
	PluginID     string   `xml:"pluginID,attr"`
	PluginName   string   `xml:"pluginName,attr"`
	PluginFamily string   `xml:"pluginFamily,attr"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: nessusmerger <input_directory> [output_file]")
		fmt.Println("  input_directory: Directory containing .nessus files to merge")
		fmt.Println("  output_file: Output merged file (default: merged_nessus_report.nessus)")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputFile := "merged_nessus_report.nessus"

	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}

	// Find all .nessus files in the input directory
	nessusFiles, err := findNessusFiles(inputDir)
	if err != nil {
		fmt.Printf("Error finding Nessus files: %v\n", err)
		os.Exit(1)
	}

	if len(nessusFiles) == 0 {
		fmt.Printf("No .nessus files found in directory: %s\n", inputDir)
		os.Exit(1)
	}

	fmt.Printf("Found %d .nessus files to merge:\n", len(nessusFiles))
	for _, file := range nessusFiles {
		fmt.Printf("  - %s\n", file)
	}

	// Count total hosts across all files for progress tracking
	fmt.Println("\nScanning files for total host count...")
	totalHosts, err := countTotalHosts(nessusFiles)
	if err != nil {
		fmt.Printf("Error counting hosts: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Total hosts to process: %d\n\n", totalHosts)

	// Merge the Nessus files with progress tracking
	mergedData, err := mergeNessusFiles(nessusFiles, totalHosts)
	if err != nil {
		fmt.Printf("Error merging Nessus files: %v\n", err)
		os.Exit(1)
	}

	// Save the merged data
	err = saveMergedReport(mergedData, outputFile)
	if err != nil {
		fmt.Printf("Error saving merged report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Successfully merged %d files into %s\n", len(nessusFiles), outputFile)
}

// findNessusFiles finds all .nessus files in the specified directory
func findNessusFiles(dir string) ([]string, error) {
	var nessusFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".nessus") {
			nessusFiles = append(nessusFiles, path)
		}

		return nil
	})

	return nessusFiles, err
}

// countTotalHosts counts the total number of hosts across all Nessus files
func countTotalHosts(files []string) (int, error) {
	totalCount := 0
	for _, file := range files {
		data, err := parseNessusFile(file)
		if err != nil {
			return 0, fmt.Errorf("error parsing %s: %v", file, err)
		}
		totalCount += len(data.Report.ReportHosts)
	}
	return totalCount, nil
}

// mergeNessusFiles merges multiple Nessus XML files into one with findings combination
func mergeNessusFiles(files []string, totalHosts int) ([]byte, error) {
	var basePolicy Policy
	hostMap := make(map[string]*ReportHost) // Track hosts by name and merge findings

	// Create progress bar
	bar := progressbar.NewOptions(totalHosts,
		progressbar.OptionSetDescription("Processing hosts"),
		progressbar.OptionSetWidth(30), // Reduced from 50 to 30
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("hosts"),
		progressbar.OptionSetTheme(progressbar.Theme{
			SaucerHead:    ">>",
			Saucer:        "=",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Printf("\n")
		}),
	)

	processedHosts := 0
	uniqueHosts := 0
	mergedHosts := 0
	totalFindings := 0

	for i, file := range files {
		data, err := parseNessusFile(file)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %v", file, err)
		}

		if i == 0 {
			// First file provides the policy template
			basePolicy = data.Policy
		}

		// Process hosts from this file
		for _, host := range data.Report.ReportHosts {
			processedHosts++

			if existingHost, exists := hostMap[host.Name]; !exists {
				// New host - add it directly
				hostCopy := host
				hostMap[host.Name] = &hostCopy
				uniqueHosts++
				totalFindings += len(host.ReportItems)
			} else {
				// Duplicate host - merge findings
				existingHost.ReportItems = append(existingHost.ReportItems, host.ReportItems...)
				mergedHosts++
				totalFindings += len(host.ReportItems)
			}

			// Update progress bar
			bar.Describe(fmt.Sprintf("Processing %s | U:%d M:%d F:%d",
				filepath.Base(file), uniqueHosts, mergedHosts, totalFindings))
			bar.Add(1)
		}
	}

	// Complete the progress bar
	bar.Finish()

	// Convert map to slice
	var allHosts []ReportHost
	for _, host := range hostMap {
		allHosts = append(allHosts, *host)
	}

	// Build the merged XML
	mergedData := NessusClientData{
		Policy: basePolicy,
		Report: Report{
			Name:        "Merged Nessus Report",
			ReportHosts: allHosts,
		},
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(&mergedData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling XML: %v", err)
	}

	// Add XML declaration
	result := []byte(xml.Header + string(output))

	fmt.Printf("\n✓ Merge complete!\n")
	fmt.Printf("  Total hosts processed: %d\n", processedHosts)
	fmt.Printf("  Unique hosts: %d\n", uniqueHosts)
	fmt.Printf("  Hosts with merged findings: %d\n", mergedHosts)
	fmt.Printf("  Total findings: %d\n", totalFindings)
	return result, nil
}

// parseNessusFile parses a single Nessus XML file using the proper structure
func parseNessusFile(filename string) (*NessusClientData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var data NessusClientData
	err = xml.Unmarshal(content, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// saveMergedReport saves the merged Nessus data to an XML file
func saveMergedReport(data []byte, filename string) error {
	// Ensure output directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the merged XML data directly
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
