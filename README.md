
# Usage

## Upgrade PowerBuilder

```
pbmanager upgrade <target>
```

## TortoiseSVN configuration

Diff command for .pbl:

```
C:\ax\dev.win.base.pbmanager\pbmanager.exe diff %base %mine --base-name %bname --mine-name %yname
```

Merge command fo .pbl:

```
C:\ax\dev.win.base.pbmanager\pbmanager.exe diff %base %mine %theirs %merged --base-name %bname --mine-name %yname --theirs-name %tname
```

## Backport PB2025 project 

```
pbmanager.exe backport {<some.pbproj>|<some.pbsln>} 
```
Only pbls and the target(s) will be created. Dlls and assets must be provided manually.
A full build is needed after the successful operation.  

# Building

## Add Icon

```
go-winres simply --icon .\choco\tools\logo.ico --arch amd64
```