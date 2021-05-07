$ErrorActionPreference = 'Stop';
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$fileLocation = Join-Path $toolsDir Setup_PBManager.exe

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  fileType      = 'exe'
  file          = $fileLocation
  softwareName  = 'PB Manager'
  fileHash      = Get-FileHash $fileLocation -Algorithm 'SHA256'
  checksum      = $fileHash.Hash
  checksumType  = $fileHash.Algorithm

  validExitCodes= @(0, 3010, 1641)
  silentArgs   = '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /SP-'
}

Install-ChocolateyPackage @packageArgs
