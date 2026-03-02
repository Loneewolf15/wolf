---
name: linkedin-post-creator
description: Create compelling LinkedIn posts written in Alex Hormozi's direct, value-packed style (maximum 150 words). Use this skill when users need to (1) research a keyword/topic via web search and generate a LinkedIn post about it, or (2) create a LinkedIn post based on their current project or work. The skill researches content, extracts key insights, and formats them in Hormozi's signature style - bold hooks, actionable insights, no fluff.
---

# LinkedIn Post Creator

## Overview

This skill helps create high-impact LinkedIn posts in Alex Hormozi's distinctive writing style - direct, actionable, and packed with value. It supports two modes: researching web content about any keyword and transforming it into posts, or creating posts about projects you're currently working on.

## Core Capabilities

### 1. Keyword-Based Post Creation

When creating a post from web research:

1. **Search** - Use `search_web` tool to find recent, high-quality content about the keyword
2. **Extract insights** - Identify the most valuable, actionable information
3. **Transform** - Convert findings into Alex Hormozi's style following the reference guide
4. **Format** - Ensure post is under 150 words, punchy, and direct

**Example user request:**
- "Create a LinkedIn post about AI automation"
- "Search for content on sales funnels and write a post"
- "Generate a post about personal branding using web research"

**Workflow:**
```
User provides keyword → Search web → Analyze top insights → 
Apply Hormozi style → Format post (≤150 words) → Present
```

### 2. Project-Based Post Creation

When creating a post about the user's current project:

1. **Understand** - Ask user about their project (or use context if available)
2. **Extract value** - Identify the core lesson, insight, or achievement
3. **Frame** - Position it as valuable business/career insight
4. **Transform** - Apply Hormozi's style principles
5. **Format** - Keep under 150 words, actionable

**Example user request:**
- "Write a LinkedIn post about this API I'm building"
- "Create a post based on my current project"
- "Turn this feature I developed into a LinkedIn post"

**What to gather:**
- What problem does the project solve?
- What did you learn?
- What results/metrics can you share?
- What mistake did you avoid/fix?

**Workflow:**
```
User describes project → Extract key insight/lesson → 
Frame for broad audience → Apply Hormozi style → 
Format post (≤150 words) → Present
```

## Alex Hormozi Style Reference

**For complete style guide, see:** [alex_hormozi_style.md](references/alex_hormozi_style.md)

**Quick style checklist:**
- ✅ Bold, attention-grabbing hook
- ✅ Direct and authoritative tone
- ✅ Concrete numbers/examples when possible
- ✅ Actionable insight or framework
- ✅ Short, punchy sentences
- ✅ No fluff or filler words
- ✅ Maximum 150 words
- ✅ Strong closing line

**Content patterns:**
- Problem → Solution → Action
- Mistake → Lesson → Framework  
- Contrarian Take → Evidence → Directive

## Output Format

When presenting the post to the user:

```markdown
## LinkedIn Post

[The actual post content, properly formatted]

---

**Word count:** [X words]
**Style elements used:** [hook type, pattern, etc.]
```

**Additional guidance:**
- If over 150 words, revise and cut ruthlessly
- Always count words and display the count
- Highlight which Hormozi patterns were used
- Maintain line breaks for readability

## Quality Standards

Every post must:
1. **Stay under 150 words** (strict limit)
2. **Lead with value** - Hook must be compelling
3. **Be actionable** - Reader should know what to do
4. **Sound like Hormozi** - Direct, confident, no BS
5. **Stand alone** - No context needed to understand

## Example Workflow

**User:** "Create a LinkedIn post about productivity hacks"

**Process:**
1. Search web for latest productivity insights
2. Find 3-5 high-value, counterintuitive tips
3. Select the most actionable/surprising one
4. Draft using "Contrarian Take → Evidence → Action" pattern
5. Cut to under 150 words
6. Review against style checklist
7. Present formatted post with word count

**Output:**
```
Most productivity advice is garbage.

Everyone says "wake up at 5am" or "meditate daily."

Here's what actually moves the needle:

Work in 90-minute blocks. Your brain operates in ultradian rhythms.
After 90 minutes, focus drops 40%.

I tracked this for 30 days:
- 90-min blocks: 6 hours of real work
- Random intervals: 3 hours of real work

Same time invested. Double the output.

Stop fighting your biology. Align with it.

---
Word count: 73 words
Style elements: Contrarian hook, concrete metrics, actionable framework
```

## Tips for Best Results

**For keyword-based posts:**
- Search for recent content (last 6 months preferred)
- Look for data, case studies, or counterintuitive findings
- Prioritize actionable insights over theory

**For project-based posts:**
- Ask clarifying questions if project context is unclear
- Focus on the business lesson, not technical details
- Extract universal insights applicable to broad audience

**When in doubt:**
- Read the [alex_hormozi_style.md](references/alex_hormozi_style.md) reference
- Cut more than you think necessary
- Make the hook stronger
- End with power
