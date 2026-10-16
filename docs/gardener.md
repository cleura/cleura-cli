# Gardener Kubernetes clusters

The `cleura gardener` command manages Cleura's Gardener-based Kubernetes.
Most commands need a **region** and **project**.
Pass `--region`/`--project-id`, set `CLEURA_REGION`/`CLEURA_PROJECT_ID`, or store them in the profile at login.
(Cloud profiles and `shoot check-name` are cloud-wide and require only a cloud ID.)

## Discover cloud-wide resources

No region or project is required:

```sh
cleura gardener cloud-profile list                  # lists machine types, Kubernetes versions, and regions
cleura gardener cloud-profile show cleuracloud      # inspects one profile in detail (supported vs deprecated versions, with expiry)
cleura gardener shoot check-name my-cluster         # check whether a shoot name is free (scripts: --exit-code → 0 free / 2 taken)
cleura gardener project bootstrap                   # a one-time operation; it enables Gardener for the project
```

## Inspect and access clusters

```sh
cleura gardener shoot get prod                                # returns full details: worker groups, maintenance, HA, networking, status
cleura gardener shoot worker-group list prod                  # lists node pools of a shoot
cleura gardener shoot kubeconfig prod > prod.kubeconfig       # creates a time-limited admin kubeconfig
cleura gardener shoot ssh-key prod -f ~/.ssh/prod-nodes.pem   # creates a node SSH private key
```

## Day-2 operations

```sh
cleura gardener shoot hibernate prod    # scales down to save cost (reversible with wake)
cleura gardener shoot wake prod         # brings the shoot back online
cleura gardener shoot reconcile prod    # runs the reconcile loop now
cleura gardener shoot maintain prod     # runs the maintenance window now
cleura gardener shoot retry prod        # retries the last failed operation
cleura gardener shoot enable-ha prod --yes                    # creates a highly-available control plane (irreversible; --yes for CI)
cleura gardener shoot ca rotate prod --stage prepare          # rotates the cluster CA (two-phase: prepare, then complete)
cleura gardener shoot ca status prod                          # returns the rotation stage and the next action in the queue
```

Non-reversible operations, like `enable-ha` and `ca rotate`, ask for confirmation and do not work non-interactively in the terminal (in a CI setting, pass the `--yes` parameter).

## Observability

Needs a running cluster:

```sh
cleura gardener shoot monitoring credentials prod             # returns Prometheus/Plutono logins (passwords are masked, use --show-secrets to reveal)
cleura gardener shoot monitoring nodes prod wg-primary        # returns per-node CPU, memory, and pods of a worker group
cleura gardener shoot monitoring worker-group prod wg-primary # returns worker-group aggregate metrics
```

Every read command outputs in JSON (`-o json`) or YAML (`-o yaml`).
