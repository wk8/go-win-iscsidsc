[![Build status](https://ci.appveyor.com/api/projects/status/github/wk8/go-win-iscsidsc?branch=master&svg=true)](https://ci.appveyor.com/project/wk8/go-win-iscsidsc/branch/master)

# go-win-iscsidsc

Golang bindings to (some of) [Windows' iSCSI Discovery Library API](https://docs.microsoft.com/en-us/windows/desktop/api/_iscsidisc/).

## Why?

If you need to manage Windows' built-in iSCSI client from a Golang code-base, and would rather avoid having to ship separate binaries and/or Powershell scripts alongside your Go-compiled executable, this library will allow you to make calls directly to Windows' API.

## How?

`go-win-iscsidsc` makes syscalls to Windows' API, and takes care of all the nitty-gritty details of converting back and forth from low-level structs to nice go structs.

## Supported functions

We currently support the following [`iscsidsc.h`](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/) functions:

* [AddIScsiConnectionW](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-addiscsiconnectionw)
* [AddIScsiSendTargetPortal](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-addiscsisendtargetportalw)
* [GetDevicesForIScsiSessionW](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-getdevicesforiscsisessionw)
* [GetIScsiSessionListW](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-getiscsisessionlistw)
* [LoginIScsiTargetW](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-loginiscsitargetw) (doesn't support custom mappings)
* [LogoutIScsiTarget](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-logoutiscsitarget)
* [RemoveIScsiSendTargetPortalW](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-removeiscsisendtargetportalw)
* [ReportIScsiSendTargetPortalsExW](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-reportiscsisendtargetportalsexw)
* [ReportIScsiTargetsW](https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-reportiscsitargetsw)

If you need more functions, please feel free to open an issue, or even better a pull request!

## Supported go versions

Our [automated builds](https://ci.appveyor.com/project/wk8/go-win-iscsidsc/branch/master) ensure that compatibility with go versions 1.9 to 1.12 included.
