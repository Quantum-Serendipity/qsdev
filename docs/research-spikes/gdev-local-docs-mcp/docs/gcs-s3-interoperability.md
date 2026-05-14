# GCS S3 Compatibility / Interoperability
- **Source**: https://docs.cloud.google.com/storage/docs/interoperability
- **Retrieved**: 2026-05-14

## API Compatibility

Cloud Storage provides XML API interoperability with S3-compatible tools and libraries. The Cloud Storage XML API is interoperable with some tools and libraries that work with services such as Amazon Simple Storage Service (Amazon S3).

## Authentication Methods

HMAC Keys: Cloud Storage uses HMAC credentials for S3 tool compatibility. Users must configure tools to use Cloud Storage HMAC keys rather than native S3 credentials.

V4 Signing: The platform supports V4 signing process authentication, allowing signed header requests to the Cloud Storage XML API using either RSA signatures or HMAC credentials.

## Endpoint Configuration

The XML API endpoint is https://storage.googleapis.com. For S3 bucket names containing dots, the gcloud CLI documentation recommends configuring path-style URLs.

## Tools & Integration

gcloud CLI: Supports S3 bucket management after adding AWS credentials to ~/.aws/credentials. Examples include listing and syncing operations between S3 and Cloud Storage buckets.

Storage Transfer Service: Enables importing data from S3, Azure Blob Storage, and HTTP/HTTPS sources, including event-driven transfers synchronized with S3 Event Notifications.

## Notable Limitations

The documentation does not exhaustively detail which specific S3 API operations are unsupported. The primary limitation mentioned concerns certificate validation errors with dotted bucket names in virtual-hosted URLs.
