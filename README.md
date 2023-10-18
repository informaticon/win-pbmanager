
# PowerBuilder Orca Interface

This is a low-level cli tool to export and import source code from/into PowerBuilder Libraries.
It is targeted to be used by [axp](https://git.informaticon.com/informaticon/dev.win.base.axp) and thus the input/output is not user firendly.

# Build

```cmd
make build
```
## Build requirements

* Python 3 **x86**
* [Visual Studio 2022 or later](https://visualstudio.microsoft.com/downloads/) (compiler and linker)
* [Nuitka Python Compiler](https://nuitka.net/doc/download.html), you can install it with `python -m pip install nuitka`

## Runtime requirements

You need to have the according PowerBuilder Utilities installed.
At Informaticon, you can install them with choco.

```cmd
choco install powerbuilder-utilities
```

# Usage


### Manual Import/Export from single PBL files
# 
.Export source from PBL to folder
[source,batch]
----
:: files are exported to the subfolder .\src\<libraryName>\
pbmanager.exe export pbl "C:\a3\grundschicht\lib\inf1.pbl"
----

.Import source from folder into PBL
[source,batch]
----
:: files are imported from the subfolder .\src\<libraryName>\
pbmanager.exe import pbl "C:\a3\grundschicht\lib\inf1.pbl"
----

### Automation / Integration

.Export source from a specific PBT (and all its PBL) to folder
[source,batch]
----
:: Export all objects
pbmanager.exe export pbt --type full "C:\a3\grundschicht\lib\a3.pbt"

:: Export only changed objects since last export
pbmanager.exe export pbt --type delta "C:\a3\grundschicht\lib\a3.pbt"
----

.Export source for all PBT files (and all their PBL) in a working tree to folder
[source,batch]
----
:: Export all objects
pbmanager.exe export folder --type full C:\a3\grundschicht\

:: Export only changed objects since last export
pbmanager.exe export folder --type delta C:\a3\grundschicht\
----
