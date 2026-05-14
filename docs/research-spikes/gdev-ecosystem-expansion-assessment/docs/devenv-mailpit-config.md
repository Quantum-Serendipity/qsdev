# devenv.sh Mailpit Service Configuration

- **Source URL**: https://devenv.sh/services/mailpit/
- **Retrieval Date**: 2026-05-14

## Configuration Options

- services.mailpit.enable — boolean, default false
- services.mailpit.package — package, default pkgs.mailpit
- services.mailpit.additionalArgs — list of strings, default []
- services.mailpit.smtpListenAddress — string, default "127.0.0.1:1025"
- services.mailpit.uiListenAddress — string, default "127.0.0.1:8025"

## Notes

- Simple, minimal configuration surface
- SMTP on port 1025, web UI on port 8025
- Modern replacement for MailHog
