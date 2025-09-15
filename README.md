# Nessus Merger

`Nessus Merger` is a command-line tool used to merge multiple Nessus (`.nessus`) scan reports into a single file. This tool intelligently combines findings for the same host found in multiple reports, consolidating all findings under a single host entry.

## Features

- Merges multiple `.nessus` XML files into a single report.
- Combines findings for the same hosts from different reports.
- Provides a progress bar to show the processing status.
- Easy-to-use command-line interface.

## Installation

To use this tool, you must have Go (version 1.22 or higher) installed on your system.

After cloning the project, you can compile the application with the following command:

```bash
go mod init nessusmerger.go
go mod tidy
go build
```

This command will create an executable file named `nessusmerger` (on macOS/Linux) or `nessusmerger.exe` (on Windows) in your project directory.

## Usage

To run the tool, you need to specify an input directory (containing the `.nessus` files) and, optionally, an output file name.

```bash
./nessusmerger <input_directory> [output_file]
```

- `<input_directory>`: The path to the directory containing the `.nessus` files to be merged.
- `[output_file]` (Optional): The name of the file where the merged report will be saved. If not specified, it defaults to `merged_nessus_report.nessus`.

### Example

To merge all `.nessus` files in the `examples/` directory and save the result as `merged_report.nessus`:

```bash
./nessusmerger examples/ merged_report.nessus
```

