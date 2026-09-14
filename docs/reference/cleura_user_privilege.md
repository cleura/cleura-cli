## cleura user privilege

View and change a user's privileges

### Synopsis

View and change what a Cleura account user may do.

A privilege is an area ('openstack', 'invoice', ...) plus a level. A level is
either 'full' or 'read' account-wide, or per-project: a set of OpenStack
projects, each with its own level. Set per-project access with --project-id and
account-wide access with --area.

The API has no privilege sub-resource, so every change here is a read, a merge
and a write of the whole user. Two changes made at the same moment can
therefore overwrite each other; the last write wins.

Account-admin rights are not part of this and cannot be changed through the
API.

```
cleura user privilege [flags]
```

### Options

```
  -h, --help   help for privilege
```

### Options inherited from parent commands

```
      --api-url string   Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string     Named cloud: public, compliant, or a private cloud's name (with --api-url) [$CLEURA_CLOUD]
      --debug            Log HTTP requests and responses to stderr (credentials redacted)
      --profile string   Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
  -q, --quiet            Suppress informational messages; errors and requested output are still shown
```

### SEE ALSO

* [cleura user](cleura_user.md)	 - Manage users in the Cleura account
* [cleura user privilege list](cleura_user_privilege_list.md)	 - List a user's privileges
* [cleura user privilege set](cleura_user_privilege_set.md)	 - Grant a user access to an area or a project
* [cleura user privilege unset](cleura_user_privilege_unset.md)	 - Remove a user's access to an area or a project

