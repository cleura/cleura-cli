# Cleura account users

`cleura user` manages the users of your **Cleura account**, that is, the identities that log in to the Control Panel and this CLI.
These are distinct from the OpenStack (Keystone) users that `cleura openstack user` manages.
The command requires `users` privileges or account-admin rights; to see your own account, use `cleura whoami`.

```sh
cleura user list                          # lists every user, with a privilege summary and 2FA status
cleura user get johndoe                   # returns full breakdown of a single user (identified by username or numeric ID)
```

## Create a user

The required password is never specified via a flag.
Instead, it is read from a no-echo prompt or from stdin, when piped:

```sh
cleura user create johndoe --email john.doe@example.org --first-name John --last-name Doe
printf '%s' "$PASSWORD" | cleura user create ci-bot --email ci@example.org    # non-interactive (CI)
```

### Edit a user

The `edit` command sends only the flags you pass, so an unset field is never overwritten:

```sh
cleura user edit johndoe --email new.address@example.org   # may need email verification
cleura user edit johndoe --first-name Jonathan
cleura user edit johndoe --set-password                    # prompts for a new password
cleura user edit ci-bot --ip-restriction 203.0.113.0/24    # replaces the whole set
cleura user edit ci-bot --clear-ip-restrictions
```

### Delete a user

Deleting a user is irreversible, so the `delete` command asks for confirmation.
On a non-interactive terminal, the `delete` command needs the `--yes` parameter.
Note that the account you are currently logged in cannot be deleted here:

```sh
cleura user delete johndoe
cleura user delete 4763 --yes
```

## Privileges

A new user starts with no access.
Privileges may be about account-wide access to an *area*, or about access to specific OpenStack *projects*.

```console
$ cleura user privilege list johndoe
AREA        LEVEL     SCOPE
invoice     read      account-wide
openstack   project   team-alpha:read, team-beta:full
```

Grant account-wide access to an area (`account`, `ai-gateway`, `application`, `invoice`, `monitoring`, `openstack`, `users`) with the `--area` parameter.
Give access to a single project with the `--project-id` parameter.
In any case, the access level is either `full` or `read`:

```sh
cleura user privilege set johndoe --area invoice --level read
cleura user privilege set johndoe --project-id team-alpha --level read
cleura user privilege set johndoe --project-id team-beta --level full
```

Each project grant leaves the others alone, so repeat the command to add more.
The `--area` parameter replaces whatever that area had, including its per-project grants.

Remove access the same way:

```sh
cleura user privilege unset johndoe --project-id team-alpha   # one project
cleura user privilege unset johndoe --area invoice            # the whole area
```

The `--project-id` parameter takes a project ID or a name.
**Project names are not unique across regions**, so a name matching more than one project is not accepted:

```plain
Error: "team-alpha" matches 2 projects; qualify it as region/name or pass the ID:
  kna1/team-alpha    bbbb2222…
  sto2/team-alpha    aaaa1111…
```

Use something like `sto2/team-alpha` or, better yet, the corresponding ID.
For a project that is not in your own project list (the API offers no account-wide project listing), pass its ID together with  `--domain-id`.

### Important notes

The API has no privilege sub-resource: a privilege change is a read, merge, and write of the whole user.
Two people changing privileges at the same moment can overwrite each other, and the last write prevails.

Account-admin rights cannot be changed through the API.
The `ADMIN` column is set to read-only for every client.
The create and edit request bodies have no such field.
Change the `ADMIN` column in the Control Panel.
