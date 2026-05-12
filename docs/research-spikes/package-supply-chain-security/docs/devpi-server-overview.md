# devpi-server: PyPI-Compatible Package Index Server

- **Source URL**: https://deepwiki.com/devpi/devpi/2-devpi-server
- **Retrieved**: 2026-05-12

## Core Functionality

devpi-server operates as a PyPI-compatible package index server providing both private repository and caching capabilities. Enables organizations to maintain internal package repositories while simultaneously serving as a caching proxy to PyPI for faster and more reliable dependency resolution.

## Proxy and Mirror Capabilities

Serves as a caching proxy to PyPI. Takes advantage of the fact that packages on PyPI are immutable: once you have a package, it can never change. Caches packages on first download.

## Index Management and Inheritance

Supports private package indexes with inheritance, enabling hierarchical index structures. Teams can create specialized indexes that inherit from parent indexes. Each user (person, project, or team) can have multiple indexes. An index using root/pypi as a parent merges PyPI packages with its own uploads.

## User Management and Authentication

- Individual user accounts with password-based authentication
- Configurable access control lists (ACLs) governing upload, deletion, and modification privileges
- Role-based permission structures for index-level access control

## Storage Infrastructure

Two complementary storage systems:
1. **KeyFS**: Transactional key-value persistence layer supporting SQLite (default) and PostgreSQL backends, ensuring atomic and consistent operations
2. **FileStore**: Dedicated storage for package files and documentation

## Replication Architecture

Supports primary-replica replication for high availability. Primary servers accept write operations while replicas synchronize data.

## REST API

Comprehensive REST API for user creation, index management, package operations, and authentication verification.

## Configuration and Deployment

Extensive configurability for diverse deployment scenarios. Modular design supporting customization through a plugin system based on pluggy. Open source (MIT license), free to use.
