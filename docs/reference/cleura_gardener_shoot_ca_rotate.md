## cleura gardener shoot ca rotate

Rotate a shoot's certificate authority

### Synopsis

Rotate a shoot's certificate authorities in two stages:
  --stage prepare    start the rotation (new CAs are issued alongside the old)
  --stage complete   finish it (the old CAs are dropped)

A rotation cannot be aborted through the CLI once prepared, and completing it
invalidates kubeconfigs issued before the rotation (re-issue them with
'cleura gardener shoot kubeconfig'). Track progress with 'shoot ca status'.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot ca rotate <shoot-name> --stage prepare|complete [flags]
```

### Examples

```
  cleura gardener shoot ca rotate prod --stage prepare
  cleura gardener shoot ca rotate prod --stage complete --yes
```

### Options

```
  -h, --help           help for rotate
      --stage string   Rotation stage: prepare or complete
  -y, --yes            Skip the confirmation prompt (required on a non-interactive terminal)
```

### Options inherited from parent commands

```
      --api-url string      Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string        Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]
      --debug               Log HTTP requests and responses to stderr (credentials redacted)
      --profile string      Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
      --project-id string   OpenStack project ID [$CLEURA_PROJECT_ID]
  -q, --quiet               Suppress informational messages; errors and requested output are still shown
      --region string       OpenStack region (e.g. sto1) [$CLEURA_REGION]
```

### SEE ALSO

* [cleura gardener shoot ca](cleura_gardener_shoot_ca.md)	 - Manage a shoot's certificate-authority rotation

