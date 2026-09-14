# Security Policy

## Supported Versions

Only the latest release and the `main` branch receive security fixes.

## Reporting a Vulnerability

Please **do not** open a public issue for security problems.

Report privately via [GitHub Security Advisories](https://github.com/castorworks/castor/security/advisories/new). Include affected version/commit, reproduction steps, and impact.

We aim to acknowledge reports within 3 business days and to ship a fix or mitigation for confirmed high-severity issues within 30 days. Reporters are credited in the advisory unless they prefer otherwise.

## Deployment Checklist

- Set `Development = false` and replace every example secret (`JwtKey`, RSA key secret, database/Redis/S3 credentials).
- Set `CASTOR_DEFAULT_ADMIN_PASSWORD` for `init-db`, then change the admin password after first login.
- Terminate TLS in front of the API and Web, and configure `TrustedProxies` to match your load balancer.
- Keep PostgreSQL, Redis and S3 on a private network.
