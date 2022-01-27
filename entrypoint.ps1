[CmdletBinding()]
param (
    [Parameter(ValueFromRemainingArguments=$true)]
    [String]
    $Passthrough
)
$ErrorActionPreference="stop"

git clone --depth=1 $Env:BLOGPOSTER_REMOTEURL /blogrepo
if (!($?)) {
    Write-error "error cloning repo"
}
cd /blogrepo && npm install
if (!($?)) {
    Write-error "npm error"
}
blogposter $Passthrough
