
# Usage

## TortoiseSVN configuration

Diff command for .pbl:

```
C:\ax\dev.win.base.pbmanager\pbmanager.exe diff %base %mine --base-name %bname --mine-name %yname
```

Merge command fo .pbl:

```
C:\ax\dev.win.base.pbmanager\pbmanager.exe diff %base %mine %theirs %merged --base-name %bname --mine-name %yname --theirs-name %tname
```

# Building

## Add Icon

```
go-winres simply --icon .\choco\tools\logo.ico --arch amd64
```