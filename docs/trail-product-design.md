# Trail: commands and user experience

## Purpose

Trail is a local, single-user CLI for keeping track of work across personal and work projects. It helps the user capture devlog notes, manage tasks and issues, and prepare a daily standup from recorded activity. The user owns the Markdown files and writes the final standup; AI is only a drafting assistant.

Trail has no accounts or database. During setup, the user chooses one data directory for all Trail projects. Each project may optionally link to a local working directory containing a Git repository. The link allows Trail to recognize the project from the current directory and, with the user's permission during the standup flow, read its Git commits. Project data stays in the Trail data directory, not in the linked repository.

## Project selection and everyday use

- Project-scoped commands use the project linked to the current directory when there is one unambiguous match. From elsewhere, the user supplies `--project <name>`; Trail does not guess.
- Project names identify projects in commands. Tags such as `work` and `personal` categorize projects; a project can have multiple tags.
- Archived projects retain their files but are absent from normal listings, `today`, and standup selection. Restoring one makes it active again.
- Commands that change Markdown report what changed. Missing projects, unavailable paths, or ambiguous project matches produce actionable errors without writing to a different project.

For example:

```text
trail init ~/trail-data
trail project add my-site --path ~/code/my-site --tag personal
trail project add payments-api --path ~/work/payments-api --tag work

# From inside ~/code/my-site:
trail log add "Reworked navigation and tested the mobile layout"
trail task add "Add keyboard navigation"
trail today

# From elsewhere:
trail task list --project my-site
```

## Commands

`[options]` below denotes the options described in the relevant section, not a literal argument. Standard help is available with `trail help [command]` or `--help`.

| Area | Command | Behavior |
| --- | --- | --- |
| Setup | `trail init <data-directory>` | Set the data directory; do not overwrite existing Trail data. |
| Setup | `trail status` | Show the data directory and the project matched to the current directory, if any. |
| Projects | `trail project add <name> [--path <directory>] [--tag <tag>]` | Create a project; a path and tags are optional. |
| Projects | `trail project list [--tag <tag>] [--all]` | List active projects, optionally filtered by tag; `--all` includes archived projects. |
| Projects | `trail project show <name>` | Show details, tags, linked path, and open-work counts. |
| Projects | `trail project rename <name> <new-name>` | Rename a project without losing its content. |
| Projects | `trail project link <name> <directory>` | Set or replace its local working-directory link. |
| Projects | `trail project unlink <name>` | Remove the link without removing any notes. |
| Projects | `trail project tag add <name> <tag>` / `trail project tag remove <name> <tag>` | Change project tags. |
| Projects | `trail project archive <name>` / `trail project restore <name>` | Hide or reactivate a project while keeping its files. |
| Devlog | `trail log add "note" [--date YYYY-MM-DD] [--project <name>]` | Append a quick note to a daily Markdown log; the date defaults to today. |
| Devlog | `trail log list [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--project <name>]` | Browse dated entries, newest first. |
| Devlog | `trail log edit [--date YYYY-MM-DD] [--project <name>]` | Open that day's Markdown log in the user's editor. |
| Tasks | `trail task add ["text"] [--due YYYY-MM-DD] [--file <filename>] [--project <name>]` | Choose a task file and add an open checkbox item; prompt for text and optional due date when omitted. |
| Tasks | `trail task list [--project <name>]` | Browse task files and select one to view and toggle its items. |
| Tasks | `trail task edit <filename> [--project <name>]` | Open an entire task file in the user's editor. |
| Issues | `trail issue add ["text"] [--file <filename>] [--project <name>]` | Choose an issue file and add an unresolved checkbox item. |
| Issues | `trail issue list [--project <name>]` | Browse issue files and select one to view and toggle its items. |
| Issues | `trail issue edit <filename> [--project <name>]` | Open an entire issue file in the user's editor. |
| Overview | `trail today [--project <name>]` | Show today's devlog and open tasks and issues from every file in the selected project. |
| Standup | `trail standup [--date YYYY-MM-DD] [--tag <tag> | --project <name>...]` | Prepare an interactive update from selected active projects. |
| Standup | `trail standup list` | List saved standups. |
| Standup | `trail standup show <date>` | Read a saved standup for the given date. |

For standups, repeating `--project` selects multiple projects. Without `--tag` or `--project`, Trail asks the user to choose **all active projects**, **projects with a tag**, or **specific projects**. It shows the resulting project list before collecting evidence. Project selection is explicit so a work update does not accidentally include personal projects.

## Markdown files and task/issue interaction

A possible data layout is:

```text
~/trail-data/
  my-site/
    project.md
    log/
      2026-09-25.md
    tasks/
      2026-09-25.md
      launch-prep.md
    issues/
      mobile-bugs.md
  standups/
    2026-09-25.md
```

Task and issue entries are ordinary Markdown checkboxes. An unchecked task is open; a checked task is done. An unchecked issue is unresolved; a checked issue is closed. An optional due date is written on a task item, rather than stored as the file's date:

```md
- [x] Set up repository
- [ ] Draft proposal @due(2026-09-25)
```

`@due(YYYY-MM-DD)` is a task-level due-date marker, not a project tag. A file's associated date describes when its collection of entries was created or organized; moving or editing a task does not change its due date. Named files must retain their associated date independently of their filename so they can appear among today's files later. Files remain readable and editable as Markdown.

### Adding to a file

`trail task add` and `trail issue add` use the same destination picker, within their respective `tasks/` or `issues/` directory:

1. If files associated with today exist, list them first. The user can append to one, create a new dated or custom-named file, or choose an older file.
2. If none exist, offer to create `YYYY-MM-DD.md`, create a custom-named file, or choose an existing file.
3. A newly created custom-named file is associated with today. A dated default file already in use is never overwritten; the user can append to it or choose another name.
4. `--file <filename>` chooses an existing file or names a new one directly, without a picker. Creation is explicit if there is a filename collision or uncertainty; Trail never silently replaces a file.
5. Trail prompts for missing item text. For tasks it also offers an optional due date; `--due` supplies that date without a prompt. Adding an issue does not prompt for a due date in v1.

Only a filename within the selected project's relevant directory can be chosen; a path cannot escape into another project or area of the filesystem. If a file cannot be read or written, Trail reports the problem without claiming the item was saved.

### Browsing and changing status

`trail task list` and `trail issue list` first show filenames and open/total counts. Selecting a file displays its checkbox items and lets the user choose one to toggle. Selecting a completed task reopens it; selecting a closed issue makes it unresolved again. For example:

```text
launch-prep.md                       1 open / 2 total

1. [x] Set up repository
2. [ ] Draft proposal               due 2026-09-25

Select an item to toggle, or press Enter to go back:
```

Toggling changes the selected checkbox in the source Markdown file while preserving its text and due-date marker. For larger edits, `task edit` and `issue edit` open the entire file in the user's editor. Entries are selected by file and displayed position, so generated task and issue IDs are not required. `trail today` and standup suggestions look through *all* task and issue files, not just files associated with today.

## Standup experience

The standup is an interactive draft, never an automatic post.

1. **Select projects.** Choose all active projects, a tag, or specific projects. `--tag` and repeated `--project` flags can preselect the scope. Trail shows the final scope before proceeding.
2. **Confirm the reporting date.** On Tuesday through Friday, the default is the previous day; on Monday, Saturday, and Sunday, the default is Friday. The user can change it interactively or pass `--date YYYY-MM-DD`. A Monday run also shows Saturday and Sunday activity separately as optional evidence, so weekend work is not silently omitted.
3. **Review the evidence.** Trail gathers devlog entries and the user's Git commits from the selected projects for the reporting date. Git activity is available only for linked directories that are Git repositories; missing or unavailable Git history does not hide devlog entries. Evidence includes its project and source, and stays separate from the draft.
4. **Draft Yesterday.** AI produces a concise summary only from the displayed recorded activity. It must not turn a commit into a claim that a task was finished unless the evidence says so. If there is no recorded activity, Trail asks the user to write Yesterday rather than inventing work. If AI is unavailable, the user can write Yesterday manually.
5. **Ask about Today.** Trail asks, “What are you planning to work on?” and suggests open tasks and unresolved issues from the selected projects. The user can accept, rewrite, add to, or ignore suggestions; their selections do not change task or issue status.
6. **Ask about Blockers.** An empty answer is valid and renders as “None.”
7. **Review the whole update.** Trail presents Yesterday / Today / Blockers for editing in the user's editor. The user's wording is final.
8. **Choose an outcome.** Copy the result, save it as Markdown under the data directory's shared `standups/` folder, or finish without saving. Saved standups are dated by the day the update is prepared; `--date` changes the activity date, not the saved standup's date. An existing saved standup for that day is not silently overwritten. Trail never posts the update to another service.

Before the first AI-assisted draft, Trail explains which devlog and commit text will be sent to the configured AI provider and lets the user proceed or write manually. AI availability is not a prerequisite for using standups. Git commits are restricted to the user's own activity; teammates' commits are not presented as the user's work.

## Boundaries and success

The initial experience is local and single-user. Project links identify local directories and provide optional Git evidence; Trail does not manage repositories, sync data, or post standups. The CLI should be useful without a linked repository or AI.

The design succeeds when the user can quickly capture notes, group tasks and issues into dated or named files, toggle individual items without losing their Markdown, see open work across those files, and produce an accurate standup that they have reviewed and chosen to copy or save.
