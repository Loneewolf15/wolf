# Wolf Language — Windows Installation Guide

## What's in this folder

| File | Purpose |
|---|---|
| `wolf-windows-amd64.exe` | The Wolf compiler binary |
| `install.bat` | **Main installer** — run this as Administrator |
| `run_wolf.bat` | Drag-and-drop launcher for quick file execution |

---

## Installing Wolf

1. **Right-click `install.bat`** and choose **"Run as administrator"**
2. The installer will:
   - Copy `wolf.exe` to `C:\Program Files\Wolf\`
   - Add it to your system `PATH` automatically
   - Add a **Start Menu** entry
   - Check for LLVM (required to compile .wolf files)
   - Create an uninstaller at `C:\Program Files\Wolf\uninstall.bat`

3. **Open a NEW terminal** (PowerShell or CMD) and verify:
   ```
   wolf --help
   ```

---

## Installing LLVM (Required)

Wolf compiles `.wolf` files to native code using LLVM. You need it installed:

**Option A — winget (recommended):**
```
winget install LLVM.LLVM
```

**Option B — Manual download:**  
https://releases.llvm.org → Download the Windows installer (`.exe`)

During LLVM installation, **check "Add LLVM to system PATH"**.

---

## Your First Wolf Program

Create `hello.wolf`:
```wolf
print("Hello from Wolf!")
```

Run it:
```
wolf run hello.wolf
```

---

## Using the Drag-and-Drop Launcher

- Drag any `.wolf` file onto `run_wolf.bat` to run it instantly
- Double-click `run_wolf.bat` for an interactive prompt

---

## Uninstalling

Run `C:\Program Files\Wolf\uninstall.bat` as Administrator,  
or delete the `C:\Program Files\Wolf\` folder and remove it from your PATH.
