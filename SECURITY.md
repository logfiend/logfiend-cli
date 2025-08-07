# Security Policy

## Reporting Security Vulnerabilities

The LogFiend team takes security seriously. We appreciate your efforts to responsibly disclose any security vulnerabilities you may find.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report security vulnerabilities by creating an ISSUE

### What to Include

When reporting a security vulnerability, please include:

1. **Description** - A clear description of the vulnerability
2. **Impact** - What could an attacker accomplish by exploiting this vulnerability?
3. **Reproduction Steps** - Step-by-step instructions to reproduce the issue
4. **Affected Versions** - Which versions of LogFiend are affected?
5. **Suggested Fix** - If you have ideas on how to fix the vulnerability
6. **Your Contact Information** - So we can follow up with questions

### Response Timeline

We commit to the following response timeline:

- **Initial Response**: Within 48 hours of receiving your report
- **Status Update**: Within 1 week with an assessment of the vulnerability
- **Resolution**: Security fixes will be prioritized and released as soon as possible
- **Public Disclosure**: We will coordinate with you on the timing of public disclosure

### Security Best Practices

LogFiend follows these security principles:

1. **Read-only by default** - Never modifies files on the host system
2. **No credential storage** - Never stores or caches authentication credentials
3. **Input sanitization** - All inputs and configurations are validated and sanitized
4. **Secure communications** - All network communications use TLS by default
5. **Minimal permissions** - Runs with minimal required permissions
6. **Transparent behavior** - All network calls are explicitly defined by user configuration

### Supported Versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

### Security Features

LogFiend includes the following security features:

- **Path traversal protection** - Prevents access to files outside the current directory
- **Input validation** - All user inputs are validated and sanitized
- **No absolute paths** - Configuration files must use relative paths
- **Credential isolation** - Credentials are read from environment variables only
- **Dry-run mode** - Test configurations without making network calls
- **Airgap support** - Can run without any network access
- **TLS enforcement** - HTTPS required for remote endpoints (HTTP only allowed for localhost)

### Known Security Considerations

- **Credential Exposure**: Always use environment variables for credentials, never hardcode them in configuration files
- **Network Monitoring**: LogFiend makes HTTP/HTTPS requests to configured SIEM endpoints - ensure these are monitored appropriately
- **File Permissions**: Output files are created with 644 permissions (readable by owner/group)
- **Configuration Security**: Keep configuration files secure as they may contain endpoint information

### Security Audit

LogFiend undergoes regular security reviews:

- **Static Analysis**: Code is analyzed using tools like `gosec` and `govulncheck`
- **Dependency Scanning**: Dependencies are regularly scanned for vulnerabilities
- **Manual Review**: Critical code paths are manually reviewed for security issues

### Responsible Disclosure

We follow responsible disclosure practices:

1. We will work with you to understand and validate the vulnerability
2. We will develop and test a fix
3. We will prepare a security advisory
4. We will coordinate the timing of the public disclosure
5. We will acknowledge your contribution (unless you prefer to remain anonymous)

### Bug Bounty

Currently, LogFiend does not offer a formal bug bounty program. However, we greatly appreciate security researchers who help improve our security posture and will acknowledge your contributions in our release notes and security advisories.

### Contact Information

For security-related questions or concerns:

- **Create Issue
- **Response Time**: 48 hours during business days

For general questions about LogFiend:

- **GitHub Issues**: For non-security related bugs and feature requests
- **Documentation**: Check our README and documentation first

---

Thank you for helping keep LogFiend and its users safe!
