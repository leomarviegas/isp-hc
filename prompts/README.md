# Prompts for multi-agent development

Files:
- prompts/gemini_prompt.md
- prompts/claude_prompt.md
- prompts/codex_prompt.md

Each file contains a ready-to-paste instruction block for the respective agent.
Use them to spawn parallel agents: Gemini (product + UX), Claude (architecture + docs), Codex (implementation + code).

Suggested workflow:
1. Run Gemini prompt to get product brief, wireframes and simulation JSONs.
2. Run Claude prompt to build HLD/LLD and runbook.
3. Run Codex prompt to generate code stubs (or use the code in this pack).
4. Cross-validate outputs automatically (each agent should validate another agent's artifact).
