# Security Policy

## Reporting a Vulnerability

Please report security vulnerabilities to **security@37signals.com**.

Do **NOT** open public GitHub issues for security vulnerabilities.

We will acknowledge receipt within 48 hours and aim to provide a fix within 90 days depending on severity.

## Credential Storage

The Fizzy CLI stores API tokens securely using your operating system's native credential storage:

| Platform | Storage |
|----------|---------|
| macOS | Keychain |
| Windows | Credential Manager |
| Linux | Secret Service (GNOME Keyring, KWallet) |

### File-based Fallback

If system keyring is unavailable (headless servers, containers), set:

```bash
export FIZZY_NO_KEYRING=1
```

Credentials will be stored as individual files in `~/.config/fizzy/credentials/`, each created with `0600` permissions.

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |
| < Latest | No       |

We only provide security fixes for the latest release. Users should upgrade promptly.
