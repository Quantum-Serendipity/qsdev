<!-- Source: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html -->
<!-- Retrieved: 2026-05-15 -->

# OWASP Logging Cheat Sheet - Audit Logging Requirements

## Essential Fields to Capture

The cheat sheet specifies logging must record "when, where, who and what" for each event:

- **When**: Log date/time (international format), event timestamp, interaction identifier
- **Where**: Application identifier/version, server address (IPv4/IPv6), service name, geolocation, code location
- **Who**: Source address, user identity (if authenticated)
- **What**: Event type, severity level, description, action, affected object, result status

## Events That Must Be Logged

Security-critical events requiring capture include:
- Authentication successes and failures
- Authorization/access control failures
- Input and output validation failures
- Session management anomalies
- Administrative actions and privilege escalation
- "Use of higher-risk functionality including...use of systems administrative privileges or access by application administrators including all actions by those users"
- Data import/export and sensitive data access
- Deserialization failures
- Network connection failures

## Log Format Recommendations

The document recommends using "standard formats over secure protocols to record and send event data" such as:
- Common Log File System (CLFS)
- Common Event Format (CEF) over syslog
- W3C Extended Log File Format

## Tamper Protection & Integrity

At rest, implement:
- "Build in tamper detection so you know if a record has been modified or deleted"
- Store logs on read-only media promptly
- Restrict and monitor all access with audit trails
- Use secure transmission protocols for untrusted networks

## Security Override Logging

The cheat sheet emphasizes logging "use of systems administrative privileges or access by application administrators including all actions by those users" and recommends monitoring "legal and other opt-ins" and suspicious business logic activities indicating flow control bypass attempts.
