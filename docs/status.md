Let’s design `trail status` as a **read-only check of Trail’s setup and current context**. It should work from any directory and never prompt or change files.

## Command contract

```text
trail status
```

- Takes **no arguments or flags** for now.
- Shows the configured data directory.
- Shows the config file path, since that’s useful for understanding and troubleshooting setup.
- Shows the project linked to the current working directory, **if there is one**. Project matching can be added when project links exist.

Unlike `init`, `status` does not repair configuration. It reports what it finds and tells the user which command can help.

## User experience

With a valid config but no matching project:

```text
Trail status

Data directory:   /Users/cedricahenkorah/trail-data
Config file:      /Users/cedricahenkorah/Library/Application Support/trail/config.json
Current project:  none
```

Later, when the current directory matches a linked project:

```text
Trail status

Data directory:   /Users/cedricahenkorah/trail-data
Config file:      /Users/cedricahenkorah/Library/Application Support/trail/config.json
Current project:  my-site
```

Important failure cases:

| State | Behavior |
| --- | --- |
| No `config.json` | Report that Trail isn’t initialized; suggest `trail init`. |
| Empty config or no data-directory setting | Report the config path; suggest `trail init` to enter the repair flow. |
| Unreadable or malformed config | Report the problem and config path; don’t pretend Trail is uninitialized. |
| Configured data directory missing or not a directory | Report the saved path and problem; don’t recreate it. |
| No project linked to the current directory | Show `Current project: none`; this is normal, not an error. |
| More than one project matches | Report the ambiguity rather than picking one. This matters once links are implemented. |

## Call stack

```text
cmd/cli/main.go: main()
  → cli.Execute()
    → Cobra checks that status has no arguments
      → runStatusCmd(cmd, args)
        → locate config.json using os.UserConfigDir()
        → read and decode the config
        → validate the saved data-directory path
        → look for a project linked to the current directory (later)
        → print status
```

The first implementation milestone is just **config → data-directory check → output**. It gives you a working command now. Later, project matching can fill in `Current project` without changing the meaning of `trail status`.

Because both `init` and `status` need to read the config, a shared config-reading function is a sensible next refactor. `init` can decide whether to set up or repair; `status` can decide what to report.
