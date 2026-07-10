## cleura gardener cloud-profile

Inspect Gardener cloud profiles (versions, machine types, images)

### Synopsis

Inspect the Gardener cloud profiles available in a cloud: the Kubernetes
versions, machine types, machine images, regions/zones and volume types you
can use when creating shoots.

This is cloud-wide: it needs a cloud (--cloud, or the profile's cloud), but no
region or project.

```
cleura gardener cloud-profile [flags]
```

### Options

```
  -h, --help   help for cloud-profile
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

* [cleura gardener](cleura_gardener.md)	 - Manage Gardener Kubernetes clusters
* [cleura gardener cloud-profile list](cleura_gardener_cloud-profile_list.md)	 - List the cloud profiles in the cloud
* [cleura gardener cloud-profile show](cleura_gardener_cloud-profile_show.md)	 - Show a cloud profile in detail

