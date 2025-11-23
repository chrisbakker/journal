

## 📓 Product Overview: Journal Web App

The Journal Web App is a modern, minimalist digital journal designed to feel like writing in a physical notebook — quiet, private, and free from clutter — while still benefiting from the convenience and structure of software.

It focuses on **daily writing**, **clean organization**, and **effortless saving**. Each day can hold multiple entries, which appear chronologically as cards. Entries can represent **meetings, notes, reflections, or other** types of writing, and each can include optional attendees and attachments.

### 🧭 Core Concept

The app is organized around **days**, not folders or tags.
Each page of the journal corresponds to a single date, and each date can contain any number of entries.

* The **calendar panel** on the left allows users to navigate quickly between days.
* The **main panel** shows the selected day’s entries.
* Each entry looks like a small journal card — with its title, text content, and type.
* Clicking an entry seamlessly switches it into an editable mode powered by the Quill rich text editor.

The result is a fluid, notebook-like experience:

* Flip to a date.
* Add a note.
* Type naturally.
* Changes save automatically.
* Flip to another date — no “save” button required.

---

## ✍️ Writing Experience

The writing interface is powered by **Quill**, providing rich text capabilities without overwhelming the user.

* Supports basic formatting: headings, bold, italic, bullet points, and code blocks.
* Clean, distraction-free editing area.
* Inline images and file attachments can be embedded directly.
* The app autosaves silently every 2 seconds of inactivity or when the editor loses focus.
* All content (text, formatting, attachments) is stored safely in a database — nothing is lost between sessions.

When not editing, entries display their saved `bodyHtml` in a simple, elegant style.
Clicking an entry transforms it into a live editor initialized from its Quill Delta JSON, so formatting and embeds are preserved exactly.

---

## 📅 Calendar & Navigation

The left-hand side of the app is a **full-month calendar**.

* The current day is highlighted.
* Days with existing entries show a small dot or subtle highlight.
* Clicking any date instantly loads that day’s entries on the right.
* You can move forward or backward month by month.

Unlike apps that require explicit “day creation,” this journal lets you **open any date freely** — the database simply stores entries tied to that date when you add one.

---

## 📂 Entry Structure

Each entry includes:

* **Title** – Short label or subject.
* **Body** – Rich text field (Quill Delta).
* **Attendees** – Comma-separated names, automatically normalized (useful for meeting notes).
* **Type** – One of `meeting`, `notes`, or `other`.
* **Attachments** – Files or images embedded in the text or stored alongside it.
* **Timestamps** – Created/updated times (UTC) and logical date fields (in user’s timezone).

Entries appear in **chronological order** within the day (oldest to newest).
Past entries are editable; no entries are “locked” automatically.

Deleting an entry performs a **soft delete** — the record remains in the database, marked as archived, so accidental deletions can be recovered later.

---

## 🗂️ Attachments & Media

Entries can include:

* Inline images displayed within the body.
* Linked attachments downloadable from within the entry.

Attachments are stored directly in the database (as binary `bytea` fields), keeping deployment simple and self-contained — no external storage service required.

Each attachment includes metadata (filename, MIME type, and size) and is referenced in the `bodyHtml` or `bodyDelta` of the parent entry.

---

## 💾 Auto-Save & Reliability

The app is designed to protect the user’s writing automatically:

* **Auto-saves every 2 seconds** after inactivity.
* Saves also occur on **blur** (when leaving a field) and **before tab close**.
* If saving fails, the entry card shows a small inline error indicator (e.g., red outline or toast message).
* When the issue resolves (e.g., reconnecting), autosave resumes seamlessly.

---

## 🕐 Timezones & Dates

The app treats each day as belonging to the **user’s local timezone** (defaulting to `America/New_York`).
All timestamps (`createdAt`, `updatedAt`) are stored in UTC for accuracy, while `year`, `month`, and `day` are derived from the user’s timezone to ensure that late-night entries appear on the correct day.

This approach ensures journal continuity even across timezones.

---

## ⚙️ Technical Philosophy

The Journal Web App emphasizes **simplicity, clarity, and evolvability**:

* Start with a single user; scale to multi-user without schema change.
* Prefer plain SQL + type-safe code generation (`sqlc`) over ORMs.
* Keep the UI minimal — no complex routing or framework overhead.
* Serve everything (API + static files) from a single Go binary.
* Optimize for reliability and low maintenance rather than features.

---

## 🌱 Planned Evolution

The app is intentionally structured to evolve through phases:

| Phase | Features                                                        |
| ----- | --------------------------------------------------------------- |
| **1** | Single-user, entries + attachments, autosave, simple calendar   |
| **2** | Search, export, theming                                         |
| **3** | Authentication, multi-user mode, logging/audit, rate limiting   |
| **4** | User timezone settings, testing, and cloud deployment readiness |

---

## 🧭 Example Use Flow

1. Open the app → Calendar shows the current month.
2. Click **today’s date** → the right pane shows existing entries (if any).
3. Click **“New Entry”** → blank Quill editor appears.
4. Add a **title**, write notes, optionally attach files.
5. Auto-save triggers as you type — no manual save required.
6. Click another entry to view or edit it.
7. Navigate days using the calendar to revisit or add notes.

---

## 🧠 Summary

The Journal Web App is a **self-contained, self-hostable daily writing system**.
It combines the **immediacy of pen and paper** with the **resilience of modern software** — no clutter, no distractions, and no external dependencies.

Its design principles:

* Minimal surface area → fewer distractions.
* Local-first mindset → runs anywhere, even offline if extended later.
* Thoughtful scalability → from personal notebook to shared knowledge system.

