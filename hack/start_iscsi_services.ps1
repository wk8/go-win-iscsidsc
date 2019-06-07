<#
.Synopsis
 Installs and starts Windows' iSCSI target and initiator services.
#>
Param(
    # if left empty, will select:
    # * if the server is already listening, then the same IP it's currently using
    # * otherwise, the first iface's IP
    # note that Windows doesn't allow 0.0.0.0 nor 127.0.0.1 here
    [Parameter(Position = 0, Mandatory=$false)] [String] $ListenIP,
    # if left empty, will select:
    # * if the server is already running, then the same port it's currently using
    # * otherwise, defaults to 3260
    [Parameter(Position = 1, Mandatory=$false)] [Int32] $ListenPort
)

. "$PSScriptRoot/.common_env.ps1"

function ensureFeatureInstalled([String]$featureName) {
    $installState = (Get-WindowsFeature -Name $featureName).InstallState
    if ($installState -eq 'Installed') {
        echo "$featureName already installed"
    } else {
        echo "Installing $featureName..."
        Install-WindowsFeature -Name $featureName -IncludeAllSubFeature -IncludeManagementTools
        echo "$featureName successfully installed"
    }
}

function ensureServiceStarted([String]$serviceName) {
    $status = (Get-Service $serviceName).Status
    if ($status -eq 'Running') {
        echo "Service $serviceName already running"
    } else {
        echo 'Starting service $serviceName...'
        Start-Service $serviceName
        (Get-Service $serviceName).WaitForStatus('Running', '00:00:30')
        echo "Service $serviceName successfully started"
    }
}

function setServerListenSettings([String]$ip, [Int32]$port) {
    $currentEndpoint = $null
    try {
        $currentPortals = (Get-IscsiTargetServerSetting).Portals

        # find the 1st portal with the desired IP
        foreach ($portal in $currentPortals) {
            if (-not $portal.Enabled) {
                continue
            }

            $portalIP = $portal.IPEndpoint.Address.ToString()
            if ((-not $ip) -or ($portalIP -eq $ip)) {
                $ip = $portalIP
                $currentEndpoint = $portal.IPEndpoint
                break
            }
        }
    } catch {
        # Get-IscsiTargetServerSetting failed, not currently listening
    }

    if ($currentEndpoint -and ((-not $port) -or ($currentEndpoint.Port -eq $port))) {
        $currentIP = $currentEndpoint.Address.ToString()
        $currentPort = $currentEndpoint.Port
        echo "iSCSITarget-Server already listening on ${currentIP}:$currentPort"
        return
    }

    if (-not $ip) {
        $ip = (Test-Connection -ComputerName (HostName) -Count 1).IPV4Address.ToString()
    }
    if (-not $port) {
        $port = 3260
    }

    echo "Setting the iSCSITarget-Server to listen on ${ip}:$port"
    Set-IscsiTargetServerSetting -IP $ip -Port $port
}

ensureFeatureInstalled 'FS-iSCSITarget-Server'
ensureServiceStarted 'WinTarget'
ensureServiceStarted 'MSiSCSI'

setServerListenSettings $ListenIP $ListenPort

echo 'iSCSI services successfully installed and started'
