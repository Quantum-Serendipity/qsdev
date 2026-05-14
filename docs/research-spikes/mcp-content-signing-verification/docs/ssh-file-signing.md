<!-- Source: https://www.agwa.name/blog/post/ssh_signatures -->
<!-- Retrieved: 2026-05-14 -->

# SSH File Signing Technical Details

## Signing Process

The `ssh-keygen -Y sign` command creates cryptographic signatures using SSH keys:

```
ssh-keygen -Y sign -f ~/.ssh/id_ed25519 -n file file_to_sign
```

Key parameters include the private key path, namespace identifier, and target file. The signature outputs to a file with `.sig` extension in a standardized format beginning with `-----BEGIN SSH SIGNATURE-----`.

## Signature Format & Structure

SSH signatures use the SSHSIG protocol, which structures the signed message as:
- Magic preamble ("SSHSIG")
- Namespace string
- Reserved field
- Hash algorithm specification
- Message hash

This differs fundamentally from traditional SSH authentication, where the first three bytes are zeros. Since "SSH" appears in position one of signatures versus zeros in SSH protocol messages, cross-protocol attacks cannot occur when reusing keys.

## Verification Process

Verification requires three components:

1. **Allowed signers file**: Maps email addresses to public keys in plain text
2. **The signature file**: The `.sig` output from signing
3. **The original file**: Read via standard input during verification

The command structure: `ssh-keygen -Y verify -f allowed_signers -I user@example.com -n file -s file.sig < file`

Successful verification returns status 0 with confirmation message including key fingerprint.

## Namespace System

Namespaces prevent signature misuse across different protocols by establishing context-specific scopes. The article recommends structuring custom namespaces like email addresses (e.g., `protocolname-v1@domain.name`) to ensure global uniqueness.

## Advantages Over PGP

Key benefits include: existing SSH infrastructure adoption, simplified key distribution via GitHub, avoiding PGP's "absurdly complex" design, and optional lightweight certificate support as an "S/MIME alternative."

## Practical Considerations

SSH signatures eliminate PGP's Web of Trust complexity while relying on trusted third parties like GitHub for key distribution -- a cleaner model than traditional cryptographic key servers.
