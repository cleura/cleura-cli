# OpenStack identity (projects, users, roles)

The `cleura openstack` command manages OpenStack (Keystone) identity — domains, projects, users, and their role assignments.
OpenStack users are **distinct from Cleura account users** (`cleura user`): they authenticate against OpenStack itself.

> **Note:** *The `cleura openstack` command does not double as a general-purpose OpenStack client, and is not meant to replace [`cleura-openstackclient`](https://github.com/cleura/cleura-openstackclient).*
>
> It deliberately covers only Keystone *identity* provisioning (projects, users, roles, and role assignments), i.e., what is covered by `cleura login` and the Cleura control plane.
> For compute, networking, storage, images, and the rest of the OpenStack API, use `cleura-openstackclient`.
> That hand-off is why the `cleura openstack` command surface is intentionally small.

Almost every command is scoped to a **domain** and needs the `--domain-id` parameter.
Cleura gives an account a domain per region, so you normally have several domains.
`cleura-cli` (trivially) auto-selects a domain only when the account has exactly one; otherwise, the `--domain-id` parameter is required.

The `domain list` and `project list` subcommands are an exception, in that they do not require the `--domain-id` parameter.
Use those subcommands to find the IDs you need:

```sh
cleura openstack domain list                       # returns domains, IDs, and their region/area
cleura openstack project list                      # returns projects you can access, grouped by region
```

## List subcommands within a domain

```sh
cleura openstack user list --domain-id <domain-id>    # returns OpenStack users in the domain
cleura openstack role list --domain-id <domain-id>    # returns assignable roles (member, load-balancer_member, etc.)
```

## Projects

The API has no delete functionality; use `--disable` when you wish to delete:

```sh
cleura openstack project create my-project --domain-id <domain-id> --description "team sandbox"   # prints the new project and ID
cleura openstack project edit <project-id> --domain-id <domain-id> --name new-name                # renames a project
cleura openstack project edit <project-id> --domain-id <domain-id> --disable                      # pseudo-deletes (turns off)
```

## Users

When using the `openstack user` subcommand, the new user's password is read from a no-echo prompt or piped stdin — never from a parameter:

```sh
cleura openstack user create alice --domain-id <domain-id>                          # prompts for the password (no echo)
printf '%s' "$PASSWORD" | cleura openstack user create svc --domain-id <domain-id>  # non-interactive (CI)
cleura openstack user delete alice --domain-id <domain-id> --yes                    # deletes an OpenStack user by name or ID (--yes for CI)
```

## Role assignments

The `role assignment` subcommand grants a user roles on a project (a role assignment is the user + project + role binding):

```sh
cleura openstack role assignment create --user alice --role member --project-id <project-id> --domain-id <domain-id>
cleura openstack role assignment create --user alice --role member,load-balancer_member --project-id <project-id> --domain-id <domain-id>
cleura openstack role assignment list   --user alice --domain-id <domain-id>    # lists projects alice can access
cleura openstack role assignment delete --user alice --role member --project-id <project-id> --domain-id <domain-id>
```

The `--user` parameter accepts a name or an ID, while the `--role` parameter takes role names.
The `--project-id` and `--domain-id` parameters accept IDs, the ones we get from the `project list` and `domain list` subcommands.

The `user delete` subcommand asks for confirmation.
It refuses to work on a non-interactive terminal — unless you add the `--yes` switch.

Every read command supports output in JSON (`-o json`) or YAML (`-o yaml`).
