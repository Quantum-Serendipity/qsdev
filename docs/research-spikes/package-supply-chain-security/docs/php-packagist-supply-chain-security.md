# PHP Supply Chain Security: Packagist Transparency Log

- **Source URL**: https://blog.packagist.com/strengthening-php-supply-chain-security-with-a-transparency-log-for-packagist-org/
- **Retrieved**: 2026-05-12

## The Transparency Log

System designed to make security-relevant events publicly visible across Packagist.org's ecosystem. Packagist hosts over 400,000 packages with 100 million daily installs.

**Tracked Events:**
- Ownership changes and source URL modifications
- Maintainer additions and removals
- Version releases and removals
- Git tag modifications
- Account security events (2FA status, password resets)

## PHP Ecosystem Private Registry Options

**Satis**: Open source, static Composer repository generator. Ultra-lightweight, static file-based. No upstream proxy capability — generates a static index of packages.

**Private Packagist**: Commercial SaaS. Offers mirroring of public packages, private package hosting, organizational package management. Developing organizational package ownership features.

**Packeton**: Open source alternative to Private Packagist, based on Satis and Packagist.

## Dependency Confusion in PHP

If a project uses a private package registry alongside Packagist, an attacker can publish a package on Packagist with the same name but higher version. Fix: explicitly configure repository priorities using the `canonical` option in composer.json.

## Ecosystem Approach

Aligns with OpenSSF's "Principles for Package Repository Security" Level 3 requirements. Funded by Sovereign Tech Agency (German government) and PHP Foundation.
