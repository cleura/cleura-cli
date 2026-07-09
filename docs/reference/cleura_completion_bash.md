## cleura completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(cleura completion bash)

To load completions for every new session, execute once:

#### Linux:

	cleura completion bash > /etc/bash_completion.d/cleura

#### macOS:

	cleura completion bash > $(brew --prefix)/etc/bash_completion.d/cleura

You will need to start a new shell for this setup to take effect.


```
cleura completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --api-url string   Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string     Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]
      --debug            Log HTTP requests and responses to stderr (credentials redacted)
      --profile string   Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
  -q, --quiet            Suppress informational messages; errors and requested output are still shown
```

### SEE ALSO

* [cleura completion](cleura_completion.md)	 - Generate the autocompletion script for the specified shell

