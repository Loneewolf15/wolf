@echo off
:: ============================================================
::  Wolf Language — Drag-and-Drop Launcher
::  Drag any .wolf file onto this to run it instantly.
:: ============================================================
title Wolf Language Runner

if "%~1"=="" (
    :: No file dropped — open an interactive Wolf shell prompt
    echo.
    echo  Wolf Language - Interactive Runner
    echo  ===================================
    echo  Usage: wolf run ^<file.wolf^>
    echo         wolf build ^<file.wolf^>
    echo         wolf --help
    echo.
    set /p "WOLF_FILE=Enter path to .wolf file (or press Enter to exit): "
    if "!WOLF_FILE!"=="" exit /b 0
    wolf run "!WOLF_FILE!"
) else (
    :: A .wolf file was dropped on this launcher
    echo  Running: %~1
    echo.
    wolf run "%~1"
)

echo.
echo  ─────────────────────────────────────
echo  Press any key to close...
pause >nul
