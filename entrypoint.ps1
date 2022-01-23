[CmdletBinding()]
param (
    [Parameter(ValueFromRemainingArguments=$true)]
    [String]
    $Passthrough
)
$ErrorActionPreference="stop"
# create config files
@{
    author=$Env:BLOGPOSTER_AUTHOR
    port="80"
    path="/blogrepo"
    username=$Env:BLOGPOSTER_USERNAME
    token=$Env:BLOGPOSTER_TOKEN
    name=$Env:BLOGPOSTER_NAME
    email=$Env:BLOGPOSTER_EMAIL
    baseurl=$Env:BLOGPOSTER_BASEURL
    remoteurl=$Env:BLOGPOSTER_REMOTEURL
    gapiconfig="/gapi.json"
} | ConvertTo-Json | Out-File -FilePath /bpconf.json

@{
    Privatekeyid=$Env:GAPI_PRIVATE_KEY_ID
    Privatekey=$Env:GAPI_PRIVATE_KEY
    Email=$Env:GAPI_EMAIL
    TokenURL=$Env:GAPI_TOKEN_URL
} | ConvertTo-Json | Out-File -FilePath /gapi.json

git clone $Env:BLOGPOSTER_REMOTEURL /blogrepo
if (!($?)) {
    Write-error "error cloning repo"
}
cd /blogrepo && npm install
if (!($?)) {
    Write-error "npm error"
}
blogposter -config="/bpconf.json" $Passthrough
