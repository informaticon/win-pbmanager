# PBManager

PBManager is a command-line interface (CLI) tool designed to streamline the management and development workflow for PowerBuilder projects.
It provides a suite of tools for upgrading, building, versioning, and managing PowerBuilder libraries and objects, all from the command line.
This makes it ideal for integration into automated build systems and modern CI/CD pipelines.

The tool is built in Go and leverages the PowerBuilder ORCA interface for its operations.

## Features

* **Backporting**: Convert PowerBuilder 2025 solution to PowerBuilder 2022R3 target.
* **Source Code Management**: Export PowerBuilder objects from PBLs into human-readable text files and import them back.
* **Version Control Integration**: A powerful diff command to compare PBL files, designed for integration with version control systems like TortoiseSVN.
* **Library Manipulation**: Delete objects from PBL files using specific names or regex patterns.
* **Project Migration**: Upgrade PowerBuilder projects to be compatible with PowerBuilder 2022R3.
* **Command-Line Builds**: Compile and build your PowerBuilder targets (.pbt) directly from the command line.

## Usage

### Backport PB2025 project

Sonvert a PB2025 solution back to a PB2022R3 workspace.
Only pbls and the target(s) will be created.

`pbmanager.exe backport <some.pbsln>`

* `--min-iter <int>`: Number of iterations through all PBL sources when errors occur. (Default `15`)

### export

Exports objects from a .pbl or .pbt file into source files.

`pbmanager export <path-to-pbl-or-pbt>`

* `-n <regex>`, `--object-name <regex>`: The name or a regex pattern of the object(s) to export. (Default: `*`)
* `-o <path>`, `--output-dir <path>`: The directory where the source files will be saved. (Default: a `src` subfolder next to the PBL/PBT file)
* `--output-encoding <encoding>`: The encoding to use for the exported files. (Default: `utf8`)
* `-s`, `--create-subdir`: Creates a sub-directory named after the PBL for the exported source files. (Default: `true`)

### import

Imports one or more source files into a specified PBL.

`pbmanager import [options] <pbl-path> <source-file-or-folder-paths...>`

* `-t <pbt-path>`, `--target <pbt-path>`: The PowerBuilder target file (.pbt) to use for the import session. If omitted, the tool will try to find it automatically.
* `-p <list>`, `--pbl-list <list>`: A comma-separated list of PBLs to import into, allowing for multi-PBL imports and resolving circular dependencies.

### delete

Removes an object from a PBL file.

`pbmanager delete <pbl-path> -n <object-name>`

* `-n <regex>`, `--object-name <regex>`: The name or regex pattern of the object(s) to delete. Required.
* `-i`, `--ignore-missing`: If set, the command will not return an error if the specified object does not exist in the PBL. (Default: `true`)

### diff

Exports the source code from two (or three) PBL files and launches a diff tool to show the differences. This is particularly useful for integrating with TortoiseSVN.

`pbmanager diff <base.pbl> <mine.pbl> [<theirs.pbl>] [<merged.pbl>]`

* `--diff-tool <path>`: Absolute path to the diff tool executable (e.g., `WinMergeU.exe`, `code.exe`). (Default: `C:/Program Files/WinMerge/WinMergeU.exe`)
* `--base-name <name>`: A descriptive name for the base file in the diff tool. (Default: `Base`)
* `--mine-name <name>`: A descriptive name for your file. (Default: `Mine`)
* `--theirs-name <name>`: A descriptive name for their file. (Default: `Theirs`)

### upgrade

Migrates a PowerBuilder project from an older version.
It applies necessary patches and performs the migration.

`pbmanager upgrade <path-to-pbt-file>`

* `--mode <mode>`: Defines the upgrade mode. Can be one of `full` (default), `patches`, `FixArf`, `FixFinDw`, `FixSqla17`.
* `--remove-exe`: If set, the existing target .exe file will be removed after migration.

### build

Compiles and builds a PowerBuilder target.

`pbmanager build <path-to-pbt-file>`

### Global Options

The following options are available for all commands:

* `--orca-version <int>`: Specifies the PowerBuilder version to use. Currently, only version `22` is supported. (Default: `22`)
* `--orca-timeout <seconds>`: Sets the timeout in seconds for PowerBuilder ORCA commands. (Default: `7200`)
* `--orca-server <address>`: The address of an Orca server to use. If not specified, a server will be started automatically.
* `--orca-apikey <key>`: The API key for the Orca server.
* `-b <path>`, `--base-path <path>`: Sets the working directory for the command. If omitted, the current directory is used.

## Building from Source

To build PBManager yourself, you need to have Go installed.

```pwsh
# Add the application icon (Optional).
# The go-winres tool is used to embed the icon into the Windows executable.
go-winres simply --icon .\\choco\\tools\\logo.ico --arch amd64

go build -o pbmanager.exe
```
