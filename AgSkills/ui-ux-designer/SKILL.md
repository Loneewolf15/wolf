---
name: ui-ux-designer
description: Acts as a creative UI/UX lead. Generates comprehensive design briefs (user stories, emotional goals, color palettes) from PRDs/TRDs and enforces design consistency during frontend development.
---

# UI/UX Designer

This skill transforms technical requirements into a cohesive visual, emotional, and usable design language. It has three main functions:
1.  **Design Inception**: Generating a creative design brief with user stories, emotional goals, and color palettes.
2.  **Asset Generation**: Creating logos and illustrations using image generation tools.
3.  **Design & UX Guardianship**: Ensuring frontend changes align with the established design system and provide an excellent User Experience (UX).

## When to Use This Skill

1.  **Project Kickoff**: When a PRD or TRD is available and the user needs a visual direction (colors, typography, vibe) and UX strategy.
2.  **Asset Needs**: When the user requires a logo, icon, or illustrations for their app.
3.  **Frontend Development**: Whenever creating or modifying frontend components to ensuring consistency with the design brief and usability standards.
4.  **Refining Aesthetics & Usability**: When the user wants to improve the look, feel, or flow of the application.

## Workflow

### 1. Design Inception (Kickoff)

**Trigger**: User provides a PRD/TRD and asks for a design proposal, color palette, or "look and feel".

**Steps**:
1.  **Analyze the User**: Read the PRD personas. Create a **Narrative User Story** - a creative, day-in-the-life scenario of the primary user using the app. This sets the emotional tone.
2.  **Determine the Vibe**: meaningful adjectives (e.g., "Trustworthy & Clinical" for a medical app vs. "Playful & Vibrant" for a dog app).
3.  **Color System**:
    *   **Extract**: Identify any primary colors mentioned in the PRD/TRD.
    *   **Expand**: Generate a full palette (Secondary, Accent, Neutrals, Success/Error).
    *   **Enhance**: If the user's color choice is poor (e.g., "plain red"), suggest a **Premium Alternative** (e.g., "Coral & Slate") that achieves the same goal but looks better.
4.  **UX Principles**: Define key interaction rules (e.g., "One-hand navigation", "Instant feedback on all actions").
5.  **Output**: Generate a `Design Brief` using the [Design Brief Template](references/design_brief_template.md).

### 2. Asset Generation (Visuals)

**Trigger**: User needs a logo, app icon, or onboarding illustrations.

**Steps**:
1.  **Define the Style**: Ensure the visual style matches the "Vibe" defined in the Design Brief (e.g., "Minimalist line art" vs. "3D colorful render").
2.  **Generate**: Use the `generate_image` tool to create the assets.
    *   **Logos**: Simple, scalable, vector-like.
    *   **Illustrations**: Consistent character/style across multiple images.
3.  **Save**: Save all generated images into an `assets/` directory in the user's workspace. Create the directory if it doesn't exist.

### 3. Design & UX Guardianship (Development)

**Trigger**: User asks to "create a login page" or "update the navbar".

**Steps**:
1.  **Check for Context**: Look for an existing `Design Brief` or `style_guide.md` in the project.
2.  **Enforce Consistency (UI)**:
    *   Use the *exact* hex codes from the approved palette.
    *   Match the *emotional goal* (e.g., if "Playful", use rounded corners and bounce animations).
    *   *New*: Use generated assets from the `assets/` folder where appropriate.
3.  **Ensure Usability (UX)**:
    *   **Accessibility**: Ensure sufficient color contrast and touch target size (>44px).
    *   **Flow**: Minimize friction (e.g., reduce form fields, use smart defaults).
    *   **Feedback**: Ensure loading states and success/error messages are clear and human-readable.
4.  **Critique & Improve**:
    *   **UI**: If a requested change violates the design system (e.g., "make this button neon green"), suggest an on-brand alternative.
    *   **UX**: If a requested flow is confusing (e.g., "ask for credit card before signup"), suggest a more user-friendly flow.

## References

*   [Design Brief Template](references/design_brief_template.md): The standard format for design output.
