# Security Policy

## Reporting a vulnerability

Please **do not open a public issue** for security problems.

Report vulnerabilities privately through GitHub's
[private vulnerability reporting](https://github.com/cleura/cleura-cli/security/advisories/new)
(the **Security** tab → *Report a vulnerability*). If you cannot use that, email
the maintainers instead of filing a public issue.

Include, as far as you can: the affected version (`cleura version`), the platform,
steps to reproduce, and the impact you observed. We aim to acknowledge reports
promptly and will keep you updated as we investigate and fix.

## Handling credentials

`cleura` stores an API token in its configuration file
(`~/.config/cleura/config.yaml`) with `0600` permissions, and redacts tokens from
`--debug` output. Tokens are short-lived. If you believe a token has been exposed,
run `cleura logout` to revoke it and `cleura login` to obtain a new one.

## Supported versions

This project is in early (`0.x`) development; only the latest release receives
security fixes.
