# `trail init` design

Let’s make the contract precise enough that you can implement it one piece at a time.

## Command contract

```text
trail init [data-directory]
```

- **One argument:** use that directory; do not prompt.
- **No argument:** ask where to put the data.
- **More than one argument:** Cobra returns an argument error.
- **Already initialized:** if the config contains a data-directory path, report it and stop; don’t replace the configuration or existing data.
- **Incomplete config:** if the config file is empty or has no data-directory path, explain the problem and offer to set one with the user's permission.

Your current `cobra.MaximumNArgs(1)` fits this contract. I’d change the help’s `Use` from `init <data-directory>` to `init [data-directory]`: square brackets communicate that it’s optional.

## Interactive experience

If the user runs `trail init` from `/home/ada/code`:

```text
Where should Trail store its data?
Use a directory in /home/ada/code? [Y/n]: y
Directory name [trail-data]:
```

Pressing Enter at the second prompt selects `/home/ada/code/trail-data`. They could instead type `my-notes`, selecting `/home/ada/code/my-notes`.

If they answer `n`:

```text
Path to your Trail data directory: ~/notes/trail
```

Then show the resolved location on success:

```text
Trail initialized. Data directory: /home/ada/notes/trail
```

A path passed as an argument should behave the same as a path entered at the prompt, minus the questions. Relative paths should be resolved against the working directory **at init time**, then stored as absolute paths. That way, later commands still work when run elsewhere.

If Trail finds an incomplete config, it first asks for permission to repair it:

```text
Trail found a config file, but it has no data directory:
  /home/ada/.config/trail/config.json

Set a data directory now? [y/N]:
```

Answering yes continues to path selection: use the supplied argument if there is one, or show the interactive location prompts if there is not. Answering no stops and leaves the config unchanged. If input is unavailable, Trail stops with an actionable error rather than replacing the file. A supplied path does not bypass the repair confirmation.

## Rules worth deciding now

- Create a new directory, or accept an **existing empty directory**. Refuse a non-empty directory so `init` never mistakes other files for Trail data or overwrites them.
- Save the selected absolute path in a small Trail config file in the user’s config directory. The data directory holds the user’s Markdown; the config tells future commands where to find it.
- A missing config starts normal setup. An unreadable config or a non-empty malformed JSON config produces an error; Trail does not treat either as a fresh setup. An empty config file or valid JSON without a data-directory path can be repaired only after explicit confirmation. Write the repaired config only after the data-directory choice is validated, without leaving a partial config on failure.
- If a configured data-directory path points to a missing directory, report that separately; don't assume the user wants to replace the setting or create a new directory.
- If a prompt receives no usable answer—for example, the path prompt is blank or input closes—return an error without creating anything.
- If directory creation or config saving fails, report the error; don’t print “initialized.”

## Call stack

Think of it as a sequence of responsibilities rather than one large `runInitCmd` function:

```text
cmd/cli/main.go: main()
  → cli.Execute()
    → Cobra checks MaximumNArgs(1)
      → runInitCmd(cmd, args)
        → read config: stop if initialized, ask to repair if incomplete, or continue if missing
        → choose path (argument or prompts)
        → expand/resolve and validate path
        → create the data directory
        → create or repair Trail's config with the chosen path
        → print success
```

Use `RunE` for `runInitCmd` so it can **return errors** to Cobra:

```go
RunE: runInitCmd

func runInitCmd(cmd *cobra.Command, args []string) error {
    // ...
}
```

I’d implement the path-selection part first and temporarily print the resolved path. Once both the argument and prompt flows select the right path, add directory creation and config saving.
