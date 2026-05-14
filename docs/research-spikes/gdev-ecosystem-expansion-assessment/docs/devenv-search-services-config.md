# devenv.sh Search Services Configuration

- **Source URLs**: https://devenv.sh/services/meilisearch/, https://devenv.sh/services/typesense/, https://devenv.sh/services/opensearch/
- **Retrieval Date**: 2026-05-14

## Meilisearch

- services.meilisearch.enable — boolean, default false
- services.meilisearch.package — package
- services.meilisearch.environment — "development" or "production", default "development"
- services.meilisearch.listenAddress — string, default "127.0.0.1"
- services.meilisearch.listenPort — uint16, default 7700
- services.meilisearch.logLevel — string, default "INFO"
- services.meilisearch.maxIndexSize — string, default "107374182400"
- services.meilisearch.noAnalytics — boolean, default true

## Typesense

- services.typesense.enable — boolean, default false
- services.typesense.package — package
- services.typesense.additionalArgs — list of strings
- services.typesense.apiKey — string, default "example"
- services.typesense.host — string, default "127.0.0.1"
- services.typesense.port — uint16, default 8108
- services.typesense.searchOnlyKey — null or string

## OpenSearch

- services.opensearch.enable — boolean, default false
- services.opensearch.package — package
- services.opensearch.settings."cluster.name" — string, default "opensearch"
- services.opensearch.settings."discovery.type" — string, default "single-node"
- services.opensearch.settings."http.port" — uint16, default 9200
- services.opensearch.settings."network.host" — string, default "127.0.0.1"
- services.opensearch.settings."plugins.security.disabled" — boolean, default true
- services.opensearch.settings."transport.port" — uint16, default 9300
- services.opensearch.extraCmdLineOptions — list of string
- services.opensearch.extraJavaOptions — list of string

## Notes

- All three search engines have native devenv.sh support
- Meilisearch: lightweight, Rust-based, fast setup, good for small-medium search
- Typesense: C++ based, simple API, good developer experience
- OpenSearch: full Elasticsearch fork, heavier but feature-complete
