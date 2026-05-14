<!-- Source: https://jedisct1.github.io/minisign/ -->
<!-- Retrieved: 2026-05-14 -->

# Minisign Technical Details

## Core Purpose
Minisign is "a dead simple tool to sign files and verify signatures" utilizing the Ed25519 public-key cryptographic system.

## Key Generation
Command: `minisign -G`

The process generates a key pair where the public key displays on screen and saves to `minisign.pub`, while the encrypted secret key stores in `~/.minisign/minisign.key`.

## Signing Operations
**Basic signing:**
```
minisign -Sm myfile.txt
```

**With trusted comments:**
```
minisign -Sm myfile.txt -t 'This comment will be signed as well'
```

Signatures output to `myfile.txt.minisig`. Batch signing of multiple files is supported.

## Verification
Two verification approaches:
```
minisign -Vm myfile.txt -P RWQf6LRCGA9i53mlYecO4IzT51TGPpvWucNSCh1CBM0QTaLn73Y7GFO3
minisign -Vm myfile.txt -p signature.pub
```

The signature file must reside in the same directory as the target file.

## Trusted Comments
The signature format includes two comment sections: an "untrusted comment" (freely modifiable) and a "trusted comment" (cryptographically bound to the signature). Trusted comments prevent downgrade attacks by embedding metadata like intended filenames, timestamps, or version numbers.

## Signify Compatibility
Minisign creates signatures verifiable by OpenBSD's `signify(1)`, but the reverse is false -- Minisign requires the trusted comment section that signify doesn't generate.

## Supported Algorithms
- **Signature:** Ed25519
- **Hashing:** Blake2b-512 (for pre-hashed signatures)
- **Checksum:** Blake2b-256
- **Key derivation:** Scryptsalsa208sha256

## Signature Format Structure
```
untrusted comment: <text>
base64(signature_algorithm || key_id || signature)
trusted_comment: <text>
base64(global_signature)
```

Legacy format uses unhashed Ed25519; current standard uses Blake2b-512 pre-hashing.

## Public Key Format
```
untrusted comment: <text>
base64(Ed || key_id || Ed25519_public_key)
```

## Secret Key Format
Encrypted structure containing algorithm identifiers, KDF salt (32 random bytes), operation/memory limits, and XOR-obfuscated key material.

## Command-Line Options Summary
- `-G`: Generate key pair
- `-S`: Sign files
- `-V`: Verify signatures
- `-R`: Recreate public key from secret key
- `-C`: Modify password protection
- `-m`: Specify file to process
- `-p`: Public key file (default: `./minisign.pub`)
- `-P`: Public key as base64 string
- `-s`: Secret key file location
- `-t`: Add trusted comment
- `-c`: Add untrusted comment
- `-W`: Disable password encryption
- `-H`: Require pre-hashed input
- `-o`: Output file content after verification
- `-q`/`-Q`: Quiet/pretty-quiet modes
- `-x`: Specify signature file
- `-f`: Force overwrite

## Memory Requirements
Pre-hashing significantly reduces memory demands during signing and verification operations.
