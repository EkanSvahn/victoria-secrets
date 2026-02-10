# Security Checklist

## Must Before Internet Exposure
- [ ] Run behind HTTPS reverse proxy with HSTS.
- [ ] Restrict Redis to private subnet and firewall rules.
- [ ] Set production `ALLOWED_ORIGIN`.
- [ ] Add API rate limiting (per IP and endpoint).
- [ ] Configure alerting on 4xx/5xx spikes.
- [ ] Enable container image and dependency scanning in CI.

## Nice Next
- [ ] Add encrypted metadata backup strategy (no plaintext content).
- [ ] Add Prometheus metrics with sensitive-field scrubbing.
- [ ] Add security.txt and disclosure policy.
- [ ] Add optional MFA-protected admin view for health/ops only.
