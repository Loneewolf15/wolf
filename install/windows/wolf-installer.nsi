; ===========================================================
;  Wolf Language — Windows GUI Installer (NSIS MUI2)
;  Produces: Wolf-Setup.exe
;  Style: Same wizard UI as Node.js / Git for Windows
; ===========================================================

; ── Modern UI 2 ─────────────────────────────────────────────
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"

; ── General ─────────────────────────────────────────────────
Name              "Wolf Language"
OutFile           "Wolf-Setup.exe"
InstallDir        "$PROGRAMFILES64\Wolf"
InstallDirRegKey  HKLM "Software\Wolf Language" "InstallDir"
RequestExecutionLevel admin
BrandingText      "Wolf Language Installer"
Unicode           true

; ── Version info embedded in .exe ───────────────────────────
VIProductVersion                 "0.1.0.0"
VIAddVersionKey "ProductName"    "Wolf Language"
VIAddVersionKey "CompanyName"    "Wolf Language Project"
VIAddVersionKey "FileDescription" "Wolf Language Compiler Installer"
VIAddVersionKey "FileVersion"    "0.1.0"
VIAddVersionKey "ProductVersion" "0.1.0"
VIAddVersionKey "LegalCopyright" "MIT License"

; ── MUI2 Interface Settings ─────────────────────────────────
!define MUI_ABORTWARNING
!define MUI_ICON                       "wolf.ico"
!define MUI_UNICON                     "wolf.ico"
!define MUI_WELCOMEFINISHPAGE_BITMAP   "wizard_banner.bmp"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP         "header.bmp"
!define MUI_HEADERIMAGE_RIGHT

; Welcome page text
!define MUI_WELCOMEPAGE_TITLE          "Welcome to Wolf Language Setup"
!define MUI_WELCOMEPAGE_TEXT           "This wizard will guide you through the installation of Wolf Language $\r$\n$\r$\nWolf is a high-performance, developer-friendly programming language that compiles to native code via LLVM.$\r$\n$\r$\nClick Next to continue."

; Finish page
!define MUI_FINISHPAGE_TITLE           "Wolf Language Setup Complete"
!define MUI_FINISHPAGE_TEXT            "Wolf has been installed on your computer.$\r$\n$\r$\nOpen a new terminal and type:$\r$\n  wolf --help$\r$\n$\r$\nOr drag any .wolf file onto the Run Wolf shortcut on your desktop."
!define MUI_FINISHPAGE_RUN             "$WINDIR\system32\cmd.exe /k wolf --help"
!define MUI_FINISHPAGE_RUN_TEXT        "Open terminal and run wolf --help"
!define MUI_FINISHPAGE_LINK            "Wolf Language Documentation"
!define MUI_FINISHPAGE_LINK_LOCATION   "https://github.com/Loneewolf15/wolf"

; ── Pages ────────────────────────────────────────────────────
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE.txt"
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; ── Language ─────────────────────────────────────────────────
!insertmacro MUI_LANGUAGE "English"

; ===========================================================
;  COMPONENTS / SECTIONS
; ===========================================================

; ── Section: Core (required) ────────────────────────────────
Section "Wolf Compiler (required)" SecCore
  SectionIn RO   ; Cannot be deselected

  SetOutPath "$INSTDIR"
  File "wolf-windows-amd64.exe"
  Rename "$INSTDIR\wolf-windows-amd64.exe" "$INSTDIR\wolf.exe"
  File "run_wolf.bat"

  ; Write install info to registry
  WriteRegStr HKLM "Software\Wolf Language" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "Software\Wolf Language" "Version"    "0.1.0"

  ; Write uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; Add to Add/Remove Programs
  WriteRegStr HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage" \
    "DisplayName" "Wolf Language"
  WriteRegStr HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage" \
    "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegStr HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage" \
    "DisplayVersion" "0.1.0"
  WriteRegStr HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage" \
    "Publisher" "Wolf Language Project"
  WriteRegStr HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage" \
    "URLInfoAbout" "https://github.com/Loneewolf15/wolf"
  WriteRegStr HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage" \
    "DisplayIcon" "$INSTDIR\wolf.exe"
  WriteRegDWORD HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage" \
    "NoModify" 1
  WriteRegDWORD HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage" \
    "NoRepair" 1

SectionEnd

; ── Section: Add to PATH ────────────────────────────────────
Section "Add wolf to system PATH" SecPath
  ; Append install dir to system PATH via registry
  ReadRegStr $0 HKLM \
    "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
  ; Only add if not already present
  ${IfNot} $0 == ""
    StrLen $1 "$INSTDIR"
    StrCpy $2 $0 $1 -$1
    ${If} $2 != "$INSTDIR"
      WriteRegExpandStr HKLM \
        "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
        "Path" "$0;$INSTDIR"
      SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" \
        /TIMEOUT=5000
    ${EndIf}
  ${EndIf}
SectionEnd

; ── Section: Start Menu ─────────────────────────────────────
Section "Start Menu Shortcuts" SecStartMenu
  CreateDirectory "$SMPROGRAMS\Wolf Language"
  CreateShortcut  "$SMPROGRAMS\Wolf Language\Wolf Terminal.lnk" \
                  "$WINDIR\system32\cmd.exe" \
                  '/k "wolf --help"' \
                  "$INSTDIR\wolf.exe" 0
  CreateShortcut  "$SMPROGRAMS\Wolf Language\Uninstall Wolf.lnk" \
                  "$INSTDIR\Uninstall.exe"
SectionEnd

; ── Section: Desktop launcher ───────────────────────────────
Section "Desktop Run Wolf launcher" SecDesktop
  CreateShortcut "$DESKTOP\Run Wolf.lnk" \
                 "$INSTDIR\run_wolf.bat" "" \
                 "$INSTDIR\wolf.exe" 0
SectionEnd

; ── Section: Install LLVM (optional) ────────────────────────
Section /o "Install LLVM toolchain (required to compile)" SecLLVM
  DetailPrint "Checking for winget..."
  nsExec::ExecToLog 'winget --version'
  Pop $0
  ${If} $0 == 0
    DetailPrint "Installing LLVM via winget (this may take a few minutes)..."
    nsExec::ExecToLog 'winget install --id LLVM.LLVM -e --accept-source-agreements --accept-package-agreements'
    Pop $0
    ${If} $0 == 0
      DetailPrint "LLVM installed successfully."
    ${Else}
      MessageBox MB_OK|MB_ICONINFORMATION \
        "LLVM could not be installed automatically.$\r$\n$\r$\nPlease install it manually from:$\r$\nhttps://releases.llvm.org$\r$\n$\r$\nDuring installation, check 'Add LLVM to system PATH'."
    ${EndIf}
  ${Else}
    MessageBox MB_OK|MB_ICONINFORMATION \
      "winget is not available on this system.$\r$\n$\r$\nPlease install LLVM manually from:$\r$\nhttps://releases.llvm.org$\r$\n$\r$\nDuring installation, check 'Add LLVM to system PATH'."
  ${EndIf}
SectionEnd

; ── Component descriptions ───────────────────────────────────
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecCore}      "The Wolf compiler (wolf.exe). Required."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecPath}      "Adds wolf to your system PATH so you can run 'wolf' from any terminal."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} "Creates a Wolf Terminal shortcut in the Start Menu."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop}   "Adds a drag-and-drop launcher shortcut to your Desktop."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecLLVM}      "Downloads and installs the LLVM toolchain (llc + clang) needed to compile .wolf files to native executables."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; ===========================================================
;  UNINSTALLER
; ===========================================================
Section "Uninstall"
  ; Remove files
  Delete "$INSTDIR\wolf.exe"
  Delete "$INSTDIR\run_wolf.bat"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir  "$INSTDIR"

  ; Remove PATH entry
  ReadRegStr $0 HKLM \
    "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"

  ; Remove start menu
  Delete "$SMPROGRAMS\Wolf Language\Wolf Terminal.lnk"
  Delete "$SMPROGRAMS\Wolf Language\Uninstall Wolf.lnk"
  RMDir  "$SMPROGRAMS\Wolf Language"

  ; Remove desktop shortcut
  Delete "$DESKTOP\Run Wolf.lnk"

  ; Remove registry entries
  DeleteRegKey HKLM "Software\Wolf Language"
  DeleteRegKey HKLM \
    "Software\Microsoft\Windows\CurrentVersion\Uninstall\WolfLanguage"

  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" \
    /TIMEOUT=5000
SectionEnd
