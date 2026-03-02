---
name: blog-creator
description: Automatically writes bi-weekly blog posts for platforms like Medium and Dev.to based on codebase progress. Use this skill when the user wants to generate a blog post about their recent work, track project progress, or manage the lifecycle of blog drafts and accepted posts within their project.
---

# Blog Creator

## Overview

The `blog-creator` skill is designed to analyze recent changes in a codebase and automatically generate engaging, technical blog posts suited for platforms like Medium and Dev.to. It monitors codebase activity, drafts bi-weekly updates, and maintains a structured directory of drafted and accepted blog posts, ensuring a consistent developer log.

## Workflow Decision Tree

### 1. Generating a New Blog Draft
When the user requests to write a new bi-weekly blog or summarize recent progress:
1. **Analyze Progress:** Investigate the codebase for recent progress. Use `git log --since="2 weeks ago"` to gather commit history, and review significant file changes or pull requests.
2. **Review Context:** Read the `README.md` or any existing project documentation to understand the broader context.
3. **Draft the Post:** Write the blog post following the guidelines in `references/blog_guidelines.md`. The tone should be engaging, technical but accessible, and highlight key learnings or challenges overcome.
4. **Save the Draft:** Save the generated blog post in the project workspace under `<project_root>/blogs/drafts/` with the naming convention `YYYY-MM-DD-blog-title.md`. Create the directory if it does not exist.
5. **Update Tracker:** Update the blog tracker file (`<project_root>/blogs/progress_tracker.md`) to reflect the newly drafted blog post.

### 2. Managing and Accepting Blogs
When the user reviews a drafted blog and accepts it:
1. **Move File:** Move the accepted blog from `<project_root>/blogs/drafts/` to `<project_root>/blogs/accepted/`. Create the directory if it does not exist.
2. **Update Tracker:** Update the status of the blog in `<project_root>/blogs/progress_tracker.md` from "Draft" to "Accepted/Ready to Publish".

### 3. Reviewing Progress
When the user asks for a summary of the blog progress:
1. **Read Tracker:** Read `<project_root>/blogs/progress_tracker.md`.
2. **Report:** Provide a concise summary of the number of drafts, accepted blogs, and the dates of the latest publications.

## Best Practices

- Always ensure the generated blog has an engaging title and a structured format (Introduction, What was built/fixed, Technical Deep Dive, Challenges, Conclusion).
- Code snippets should be included where they illustrate a complex or interesting problem solved.
- Regularly check `references/blog_guidelines.md` for structural patterns and platform-specific formatting tips.
