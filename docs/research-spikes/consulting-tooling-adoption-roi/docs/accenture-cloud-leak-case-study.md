<!-- Source: https://www.upguard.com/breaches/cloud-leak-accenture -->
<!-- Retrieved: 2026-03-20 -->

# Accenture Cloud Leak Case Study

## Discovery Details
On September 17, 2017, UpGuard's Chris Vickery identified four unsecured Amazon Web Services S3 buckets configured for public access. The buckets were secured the following day after notification.

## Exposed Data
The four buckets ("acp-deployment," "acpcollector," "acp-software," and "acp-ssl") contained:

- **API credentials and authentication keys** for Accenture's Identity API
- **Master access keys** for AWS Key Management Service in plaintext
- **Nearly 40,000 plaintext passwords** from database backups
- **Private signing keys and certificates** potentially enabling traffic decryption
- **VPN keys** used in production for Accenture's private network
- **Customer credentials** from Accenture clients
- **Google and Azure account credentials**
- **Enstratus access keys** for cloud infrastructure management
- **Internal email information** and ASGARD database details

## Scale
The largest bucket, "acp-software," contained 137 GB of data including extensive database dumps.

## Impact
The exposure potentially threatened 94 of the Fortune Global 100 and over three-quarters of the Fortune Global 500—all customers of Accenture Cloud Platform. Competent threat actors could have impersonated Accenture, accessed client systems, or exploited password reuse vulnerabilities.

## Response
Accenture took immediate action securing the buckets once notified, preventing further unauthorized access.
