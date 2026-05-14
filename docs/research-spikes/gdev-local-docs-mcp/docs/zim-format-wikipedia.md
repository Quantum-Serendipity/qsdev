# ZIM File Format - Wikipedia
> Source: https://en.wikipedia.org/wiki/ZIM_(file_format)
> Retrieved: 2026-05-14

## Core Definition
The ZIM format is an open file format that stores website content for offline usage. Primarily designed for storing Wikipedia and Wikimedia project contents.

## Nomenclature
ZIM stands for "Zeno IMproved" as it superseded the earlier Zeno format.

## Compression Methods
Since 2021, the library defaults to Zstandard file compression and also supports LZMA2, as implemented by the XZ Utils library. The compression ratio can be up to 3x with almost all of the space savings taking place within the clusters.

## Content Storage
The format handles articles, full-text search indices and auxiliary files from Wikipedia and related projects.

Note: Wikipedia article is thin on technical details. The authoritative source is the openZIM wiki specification (blocked by Anubis during fetch attempts).
