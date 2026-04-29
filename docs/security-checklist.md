# Security Checklist

## Must Before Internet Exposure
- [x] Run behind HTTPS reverse proxy with HSTS.
- [ ] Restrict Redis to private subnet and firewall rules.
- [ ] Set production `ALLOWED_ORIGINS`.
- [x] Add API rate limiting (per IP and endpoint).
- [ ] Configure alerting on 4xx/5xx spikes.
- [x] Enable dependency scanning in CI.
- [x] Enable container image scanning in CI.

## Nice Next
- [ ] Add encrypted metadata backup strategy (no plaintext content).
- [ ] Add Prometheus metrics with sensitive-field scrubbing.
- [ ] Add security.txt and disclosure policy.
- [ ] Add optional MFA-protected admin view for health/ops only.
