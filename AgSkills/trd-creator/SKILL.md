---
name: trd-creator
description: Generates high-quality Technical Requirements Documents (TRD) based on existing requirements or PRDs. Use this skill when the user wants to create a TRD, convert a PRD to a TRD, or improve an existing TRD following industry benchmarks.
---

# TRD Creator

This skill helps users create comprehensive and industry-standard Technical Requirements Documents (TRD).

## When to Use This Skill

Use this skill when the user:
1.  Requests to create a TRD from scratch.
2.  Provides a PRD and wants a corresponding TRD.
3.  Wants to improve or standardize an existing TRD.
4.  Needs to document system architecture, API design, or data models for a project.

## Workflow

1.  **Analyze Inputs**:
    *   If the user provides a PRD, read it thoroughly to understand the features and functional requirements.
    *   If the user provides a loose description, ask clarifying questions to gather necessary technical details (e.g., tech stack preference, scale, security needs).
    *   If the user provides an existing TRD, identify gaps compared to the standard template.

2.  **Generate/Update TRD**:
    *   Use the [TRD Template](references/trd_template.md) as the structure.
    *   Fill in each section with specific, actionable technical details.
    *   **Crucial**: Do not just copy the PRD. Translate functional requirements into technical specifications (e.g., "User logs in" (PRD) -> "POST /api/login with JWT authentication" (TRD)).

3.  **Refine**:
    *   Ensure diagrams (Mermaid) are included for architecture and data flow.
    *   Verify that NFRs (Non-Functional Requirements) like performance and security are strictly defined.

## References

*   [TRD Template](references/trd_template.md): The standard structure for the TRD.
