@echo off
:: ============================================================
::  Wolf Language — Windows Installer
::  Run as Administrator
:: ============================================================
setlocal EnableDelayedExpansion

:: ── Config ──────────────────────────────────────────────────
set "WOLF_VERSION=dev"
set "INSTALL_DIR=C:\Program Files\Wolf"
set "WOLF_EXE=%~dp0wolf-windows-amd64.exe"
set "START_MENU=%ProgramData%\Microsoft\Windows\Start Menu\Programs\Wolf Language"

:: ── Banner ──────────────────────────────────────────────────
echo.
echo  ██╗    ██╗ ██████╗ ██╗     ███████╗
echo  ██║    ██║██╔═══██╗██║     ██╔════╝
echo  ██║ █╗ ██║██║   ██║██║     █████╗
echo  ██║███╗██║██║   ██║██║     ██╔══╝
echo  ╚███╔███╔╝╚██████╔╝███████╗██║
echo   ╚══╝╚══╝  ╚═════╝ ╚══════╝╚═╝  Language Installer
echo.
echo  Version : %WOLF_VERSION%
echo  Target  : %INSTALL_DIR%
echo.

:: ── Admin check ─────────────────────────────────────────────
net session >nul 2>&1
if %errorLevel% NEQ 0 (
    echo  [ERROR] Please run this installer as Administrator.
    echo  Right-click install.bat and choose "Run as administrator".
    echo.
    pause
    exit /b 1
)

:: ── Check wolf binary exists next to installer ──────────────
if not exist "%WOLF_EXE%" (
    echo  [ERROR] wolf-windows-amd64.exe not found next to install.bat
    echo  Please place both files in the same folder and try again.
    echo.
    pause
    exit /b 1
)

:: ── Create install directory ────────────────────────────────
echo  [1/5] Creating install directory...
if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%"
    if !errorLevel! NEQ 0 (
        echo  [ERROR] Could not create %INSTALL_DIR%
        pause
        exit /b 1
    )
)

:: ── Copy wolf binary ────────────────────────────────────────
echo  [2/5] Installing wolf.exe...
copy /Y "%WOLF_EXE%" "%INSTALL_DIR%\wolf.exe" >nul
if %errorLevel% NEQ 0 (
    echo  [ERROR] Failed to copy wolf.exe
    pause
    exit /b 1
)

:: ── Copy run_wolf.bat launcher ──────────────────────────────
set "BAT_LAUNCHER=%~dp0run_wolf.bat"
if exist "%BAT_LAUNCHER%" (
    copy /Y "%BAT_LAUNCHER%" "%INSTALL_DIR%\run_wolf.bat" >nul
)

:: Copy README if present
set "README=%~dp0README.md"
if exist "%README%" (
    copy /Y "%README%" "%INSTALL_DIR%\README.md" >nul
)

:: ── Add to System PATH ──────────────────────────────────────
echo  [3/5] Adding Wolf to system PATH...
:: Read current PATH
for /f "tokens=2,*" %%A in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH 2^>nul') do set "CURRENT_PATH=%%B"

:: Check if already in PATH
echo !CURRENT_PATH! | findstr /I /C:"%INSTALL_DIR%" >nul 2>&1
if %errorLevel% NEQ 0 (
    reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" ^
        /v PATH /t REG_EXPAND_SZ ^
        /d "!CURRENT_PATH!;%INSTALL_DIR%" /f >nul
    echo     Added %INSTALL_DIR% to PATH.
) else (
    echo     Already in PATH, skipping.
)

:: ── Start Menu shortcut ─────────────────────────────────────
echo  [4/5] Creating Start Menu entry...
if not exist "%START_MENU%" mkdir "%START_MENU%"

:: Create a .bat shortcut wrapper in Start Menu
set "SM_BAT=%START_MENU%\Wolf Terminal.bat"
(
    echo @echo off
    echo title Wolf Language
    echo cmd /k "wolf --help"
) > "%SM_BAT%"

:: Write a README shortcut
set "SM_README=%START_MENU%\Wolf Docs (README).bat"
(
    echo @echo off
    echo start notepad "%INSTALL_DIR%\README.md"
) > "%SM_README%"

:: ── Check for LLVM/clang ────────────────────────────────────
echo  [5/5] Checking for LLVM toolchain...
where llc >nul 2>&1
set "LLVM_OK=%errorLevel%"

where clang >nul 2>&1
set "CLANG_OK=%errorLevel%"

:: ── Broadcast PATH change (so new terminals see it) ─────────
:: Send WM_SETTINGCHANGE to all windows
powershell -NoProfile -Command ^
  "[System.Environment]::SetEnvironmentVariable('PATH', [System.Environment]::GetEnvironmentVariable('PATH','Machine'), 'Machine')" >nul 2>&1

:: ── Create uninstaller ──────────────────────────────────────
set "UNINSTALL=%INSTALL_DIR%\uninstall.bat"
(
    echo @echo off
    echo net session ^>nul 2^>^&1
    echo if %%errorLevel%% NEQ 0 ^(
    echo     echo Run as Administrator to uninstall.
    echo     pause
    echo     exit /b 1
    echo ^)
    echo echo Removing Wolf Language...
    echo rmdir /S /Q "%INSTALL_DIR%"
    echo :: Remove from PATH
    echo for /f "tokens=2,*" %%%%A in ^('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH'^) do set "P=%%%%B"
    echo setlocal EnableDelayedExpansion
    echo set "P=!P:;%INSTALL_DIR%=!"
    echo reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH /t REG_EXPAND_SZ /d "!P!" /f ^>nul
    echo rmdir /S /Q "%START_MENU%" 2^>nul
    echo echo Wolf Language has been uninstalled.
    echo pause
) > "%UNINSTALL%"

:: ── Summary ─────────────────────────────────────────────────
echo.
echo  ════════════════════════════════════════════════
echo   Wolf Language installed successfully!
echo  ════════════════════════════════════════════════
echo.
echo   Location : %INSTALL_DIR%\wolf.exe
echo   Start Menu: %START_MENU%
echo   Uninstall : %INSTALL_DIR%\uninstall.bat
echo.

if %LLVM_OK% NEQ 0 (
    echo  ⚠  LLVM not found in PATH.
    echo     Wolf needs 'llc' and 'clang' to compile .wolf files.
    echo.
    echo     Install LLVM from: https://releases.llvm.org
    echo     Or run:  winget install LLVM.LLVM
    echo.
) else (
    if %CLANG_OK% NEQ 0 (
        echo  ⚠  clang not found but llc was found. Install clang too:
        echo     winget install LLVM.LLVM
        echo.
    ) else (
        echo  ✓  LLVM toolchain detected. Wolf is fully ready!
        echo.
    )
)

echo   Open a NEW terminal window and run:
echo     wolf --help
echo     wolf run hello.wolf
echo.
echo  ════════════════════════════════════════════════
echo.
pause
