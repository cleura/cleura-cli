## cleura gardener shoot ca

Manage a shoot's certificate-authority rotation

### Synopsis

Manage the two-phase certificate-authority (CA) rotation of a shoot cluster.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot ca [flags]
```

### Options

```
  -h, --help   help for ca
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

* [cleura gardener shoot](cleura_gardener_shoot.md)	 - Manage shoot clusters
* [cleura gardener shoot ca rotate](cleura_gardener_shoot_ca_rotate.md)	 - Drive a two-phase CA rotation
* [cleura gardener shoot ca status](cleura_gardener_shoot_ca_status.md)	 - Show a shoot's CA rotation stage

