---
name: ionic-angular-designer
description: Design and build modern, high-quality Ionic Angular applications using Standalone components and bold aesthetics. Use this skill when the user wants to creating mobile apps, UI components, or layouts with the Ionic Framework and Angular. It enforces best practices (Standalone API) and distinctive design choices.
---

# Ionic Angular Designer

## Overview

This skill specializes in creating production-grade Ionic Angular applications that combine:
1.  **Modern Architecture**: Strict adherence to the Standalone Component API (Angular 17+ / Ionic 7.5+).
2.  **Bold Design**: Distinctive, non-generic user interface design that elevates the standard Ionic look.

## Usage Process

### 1. Technical Foundation (The "Skeleton")

**Rule #1: Standalone Everything.**
*   NEVER generate traditional `NgModules`.
*   ALWAYS use `standalone: true` in your `@Component` decorator.
*   imports must come from `@ionic/angular/standalone` (e.g., `IonContent`), NOT `@ionic/angular`.
*   See [references/standalone_patterns.md](references/standalone_patterns.md) for boilerplate.

### 2. Aesthetic Direction (The "Skin")

Ionic apps often look "default" (standard iOS/Material styles). **Avoid this.** To create a high-quality app:

*   **Typography**: Don't just use the system font. Import a distinctive font (e.g., from Google Fonts) and apply it via CSS variables (`--ion-font-family`).
*   **Color Palette**: Define a semantic, bold color palette in `global.scss` or `theme/variables.scss`.
    *   Avoid default Ionic Blue (`#3880ff`). Pick something memorable.
    *   Use CSS variables: `--ion-color-primary`, `--ion-color-primary-shade`, etc.
*   **Custom Styling**:
    *   Use **Shadow Parts** (`::part`) to style inside Ionic components.
    *   Use **CSS Utilities** (padding, flex) or Tailwind (if requested) for layout.
    *   Add **Micro-interactions**: Use `ion-ripple-effect` explicitly or Angular Animations for state changes.

### 3. Implementation Workflow

When asked to design/build an Ionic page or component:

1.  **Define the Vibe**: Ask "Is this a playful consumer app? A stark financial tool? A warm social platform?"
2.  **Select Components**: Choose the right Ionic UI blocks (`ion-card`, `ion-modal`, `ion-segment`).
3.  **Write Code**:
    *   Create the standalone component structure.
    *   Register necessary icons (`addIcons`).
    *   Apply the "Bold Design" principles (custom CSS, fonts, spacing).

## Example: A Modern "Profile" Page

**Bad (Generic):**
*   Default blue header.
*   Standard list items.
*   System font.

**Good (Designed):**
*   Header: Transparent with a large, bold avatar overlapping the background.
*   Typography: "Outfit" or "Space Grotesk" for headers.
*   Colors: Deep purple primary with neon green accents (or whatever fits the specific vibe).
*   Interaction: Smooth reveal animations on list items.

## Reference

**Standalone Components Guide**: [references/standalone_patterns.md](references/standalone_patterns.md)
*   Check this file for exact syntax on `bootstrapApplication`, imports, and routing.

**Design Principles**:
*   **Be Bold**: Make specific choices.
*   **Be Consistent**: Use CSS Variables.
*   **Be Modern**: Use Standalone APIs.
