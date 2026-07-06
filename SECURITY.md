# Security Policy

BindKit is a starter kit for MCP servers. Please treat vulnerabilities in auth,
billing, metering, rate limiting, SSRF protections, or MCP request handling as
security-sensitive.

## Reporting a vulnerability

Email security reports to `security@bindfort.com`.

Please include:

- affected version or commit;
- reproduction steps;
- expected and actual behavior;
- impact assessment;
- whether the report can be publicly credited.

Do not publish exploit details, secrets, customer data, or weaponized proof of
concepts before coordination.

## Supported versions

Until the first tagged public release, security fixes target `main`.

After `v1.0.0`, the project will document supported release branches here.

## Scope

In scope:

- auth bypasses;
- JWT/JWKS validation flaws;
- Stripe webhook verification flaws;
- quota, rate-limit, or metering bypasses;
- SSRF bypasses in bundled tools;
- unsafe MCP request parsing or transport behavior;
- secret leakage in logs.

Out of scope:

- attacks requiring modification of local source files;
- dependency findings without a reachable impact path;
- missing production hardening in user-created tools built from the template.
