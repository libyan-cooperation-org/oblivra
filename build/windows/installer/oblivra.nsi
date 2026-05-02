; OBLIVRA Windows Installer (NSIS)
; ==================================
; Builds OBLIVRA-Setup-vX.Y.Z.exe — a single-file installer that drops:
;   - Headless server: oblivra-server.exe (web UI on http://localhost:8080)
;   - Forwarder:       oblivra-agent.exe
;   - CLI:             oblivra-cli.exe
;   - Verifier:        oblivra-verify.exe
;   - Smoke / soak:    oblivra-smoke.exe, oblivra-soak.exe
;   - Migrate:         oblivra-migrate.exe
;
; OBLIVRA is web-only — the Wails desktop shell was retired. Operators
; reach the UI via browser at the address oblivra-server.exe binds to.
;
; Build with:
;   makensis /DVERSION=0.1.0-beta1 build/windows/installer/oblivra.nsi
;
; The output drops at build/windows/installer/OBLIVRA-Setup-<VERSION>.exe.

!ifndef VERSION
  !define VERSION "0.1.0-beta1"
!endif

!define APP_NAME      "OBLIVRA"
!define APP_PUBLISHER "OBLIVRA Contributors"
!define APP_URL       "https://github.com/libyan-cooperation-org/oblivra"
!define APP_EXE       "oblivra-server.exe"
!define UNINST_KEY    "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"

Name           "${APP_NAME} ${VERSION}"
OutFile        "OBLIVRA-Setup-${VERSION}.exe"
InstallDir     "$PROGRAMFILES64\${APP_NAME}"
InstallDirRegKey HKLM "Software\${APP_NAME}" "InstallDir"
RequestExecutionLevel admin
SetCompressor  /SOLID lzma
ShowInstDetails show
ShowUninstDetails show
Unicode true

; ---- Includes ------------------------------------------------------------

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"

!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\..\LICENSE"
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\${APP_EXE}"
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

; ---- Version metadata (right-click → Properties on the .exe) ------------

VIProductVersion "0.1.0.0"
VIAddVersionKey "ProductName"     "${APP_NAME}"
VIAddVersionKey "ProductVersion"  "${VERSION}"
VIAddVersionKey "FileVersion"     "0.1.0.0"
VIAddVersionKey "CompanyName"     "${APP_PUBLISHER}"
VIAddVersionKey "FileDescription" "${APP_NAME} sovereign log-driven security platform installer"
VIAddVersionKey "LegalCopyright"  "Apache License 2.0 — see LICENSE"

; ---- Install -------------------------------------------------------------

Section "OBLIVRA Server + tools" SecCore
  SectionIn RO
  SetOutPath "$INSTDIR"
  File "..\..\bin\oblivra-server.exe"
  File "..\..\bin\oblivra-cli.exe"
  File "..\..\bin\oblivra-verify.exe"
  File "..\..\bin\oblivra-migrate.exe"
  File "..\..\bin\oblivra-smoke.exe"
  File "..\..\bin\oblivra-soak.exe"
  File "..\..\..\LICENSE"
  File "..\..\..\README.md"
  File "..\..\..\SECURITY.md"

  ; Sigma rule pack — operators expect rules to live next to the binary.
  SetOutPath "$INSTDIR\sigma"
  File /r "..\..\..\sigma\*.yml"

  WriteRegStr HKLM "Software\${APP_NAME}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "Software\${APP_NAME}" "Version"    "${VERSION}"

  WriteRegStr HKLM "${UNINST_KEY}" "DisplayName"     "${APP_NAME} ${VERSION}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon"     "$INSTDIR\${APP_EXE}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion"  "${VERSION}"
  WriteRegStr HKLM "${UNINST_KEY}" "Publisher"       "${APP_PUBLISHER}"
  WriteRegStr HKLM "${UNINST_KEY}" "URLInfoAbout"    "${APP_URL}"
  WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
  WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1

  WriteUninstaller "$INSTDIR\Uninstall.exe"
SectionEnd

Section "Forwarder agent" SecAgent
  SetOutPath "$INSTDIR"
  File "..\..\bin\oblivra-agent.exe"
SectionEnd

Section "Start menu shortcuts" SecStartMenu
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\${APP_NAME} server.lnk"    "$INSTDIR\oblivra-server.exe"
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\Open web UI.lnk"           "http://localhost:8080/"
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\Uninstall ${APP_NAME}.lnk" "$INSTDIR\Uninstall.exe"
SectionEnd

; ---- Section descriptions ------------------------------------------------

LangString DESC_SecCore      ${LANG_ENGLISH} "Headless server (web UI), CLI, offline verifier, smoke/soak harnesses (required)."
LangString DESC_SecAgent     ${LANG_ENGLISH} "oblivra-agent log forwarder. Install on every host that ships logs."
LangString DESC_SecStartMenu ${LANG_ENGLISH} "Start menu shortcuts for the desktop app and headless server."

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecCore}      $(DESC_SecCore)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecAgent}     $(DESC_SecAgent)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} $(DESC_SecStartMenu)
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; ---- Uninstall -----------------------------------------------------------

Section "Uninstall"
  Delete "$INSTDIR\oblivra-server.exe"
  Delete "$INSTDIR\oblivra-agent.exe"
  Delete "$INSTDIR\oblivra-cli.exe"
  Delete "$INSTDIR\oblivra-verify.exe"
  Delete "$INSTDIR\oblivra-migrate.exe"
  Delete "$INSTDIR\oblivra-smoke.exe"
  Delete "$INSTDIR\oblivra-soak.exe"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\README.md"
  Delete "$INSTDIR\SECURITY.md"
  Delete "$INSTDIR\Uninstall.exe"

  RMDir /r "$INSTDIR\sigma"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME} server.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Open web UI.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Uninstall ${APP_NAME}.lnk"
  RMDir "$SMPROGRAMS\${APP_NAME}"

  DeleteRegKey HKLM "Software\${APP_NAME}"
  DeleteRegKey HKLM "${UNINST_KEY}"

  ; Note: we deliberately do NOT delete user data
  ; (%LOCALAPPDATA%\OBLIVRA, vault file, audit.log, warm parquet files).
  ; Those are operator-controlled artefacts and removing them silently
  ; would destroy evidence. Operators who want a full wipe can rm -rf
  ; %LOCALAPPDATA%\OBLIVRA themselves.
SectionEnd
