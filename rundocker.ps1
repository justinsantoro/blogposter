[CmdletBinding()]
param (
    [Parameter()]
    [String]
    $Port,
    [Parameter()]
    [String]
    $EnvFile,
    [String]
    $RunArgs,
    [Parameter(ValueFromRemainingArguments=$true)]
    [String]
    $Passthrough
)
$ErrorActionPreference="stop"

docker image inspect blogposter
if (!($?)) {
    # image doesnt exist. build it
    docker build -t blogposter .
    if (!($?)) {
        Write-Error "error building image"
    }
}

docker run `
    -p ${Port}:80 `
    --env-file $EnvFile $RunArgs `
    blogposter $Passthrough
