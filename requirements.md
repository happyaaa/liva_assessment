# **LIVA AI ASSESSMENT**

TIME LIMIT: ~4 hours (the full recording including speaking, writing, coding)

---

## **Problem**

Build a system that tracks user earnings from audio recordings.

**RULES**

- Users participate in recordings and earn $1/hour (pro-rated to the minute, round down to nearest minute)
- When a recording ends, participants are credited
- Users can withdraw earnings anytime
- Balance can never go negative

**FRAUD**

- A user in two overlapping recordings = fraud
- Fraud → $0 for that user on BOTH recordings
- Other participants are paid normally

---

## **API**

```

POST /recording/end   → credit earnings, detect fraud

GET  /balance/:userId → return balance

POST /withdraw        → withdraw funds

```

Stack: TypeScript, Go, or Python

In-memory storage. You decide the request/response formats.

---

## **Requirements**

- Fraud detection must be O(log n) per user (n = recordings per user)
- Concurrent requests must not corrupt data
- Write tests demonstrating correct fraud handling

---

## **Submission**

Send:

1. Working code with tests (github repo) No deployment needed.

2. Screen recording of your full session (with voice narration where appropriate)

3. Brief write-up if you didn't go over them in the recording:

- What edge cases did you discover? How did you handle them?
- What could break at 10k recordings/min? How would you fix it?

---

## **Recording Guidelines**

- Record your entire screen + voice for the full session
- Start a timer at the beginning
- Narrate your thinking as you work (silence during reading is fine)
- Do not work outside the recorded session
- NOT ALLOWED: AI coding agents like Claude Code, Codex, Cursor, etc. We will use lots of AI at work, but for this assessment, we want to see your raw problem solving process
- ALLOWED: Auto-completion, internet search, reading documentation, using AI to look up concepts (not write code)

---

## What we’re looking for

- Clarity in thinking (MOST IMPORTANT)
- Clean, readable code
- Detail oriented, can think of edge cases thoughtfully
- Reasonable design decisions