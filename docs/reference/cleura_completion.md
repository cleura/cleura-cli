## cleura completion

Generate the autocompletion script for the specified shell

### Synopsis

Generate the autocompletion script for cleura for the specified shell.
See each sub-command's help for details on how to use the generated script.


### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --api-url string      Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string        Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]
      --debug               Log HTTP requests and responses to stderr (credentials redacted)
  -o, --output string       Output format: table, json, yaml (default "table")
      --profile string      Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
      --project-id string   OpenStack project ID [$CLEURA_PROJECT_ID]
  -q, --quiet               Suppress informational messages; errors and requested output are still shown
      --region string       OpenStack region (e.g. sto1) [$CLEURA_REGION]
```

### SEE ALSO

* [cleura](cleura.md)	 - Command-line interface for Cleura Cloud
* [cleura completion bash](cleura_completion_bash.md)	 - Generate the autocompletion script for bash
* [cleura completion fish](cleura_completion_fish.md)	 - Generate the autocompletion script for fish
* [cleura completion powershell](cleura_completion_powershell.md)	 - Generate the autocompletion script for powershell
* [cleura completion zsh](cleura_completion_zsh.md)	 - Generate the autocompletion script for zsh

