---
source: https://raw.githubusercontent.com/sth/sodata/master/README.md
retrieved: 2026-05-14
---

# Stack Exchange Data Dump Import Tool (sodata)

## Import Process
Converts XML data files into databases through three specialized importers:
- **sqliteimport**: Creates SQLite3 databases (default filename: `dump.db`)
- **pgimport**: Populates PostgreSQL databases with connection string specifications
- **csvimport**: Generates CSV files from XML source data

The import mechanism relies on libexpat to parse the input files and requires running these tools in a directory containing extracted XML dumps.

## Table and Field Structure
Creates database tables that mirror the XML file names, resulting in tables such as `posts`, `comments`, and `users`. "The column names in those tables correspond to the attribute names in the XML."

For precise schema definitions, the documentation references `soschema.hpp`, which "defines the tables and columns that get imported."

## Key Technical Details
**PostgreSQL Import Options:**
- Standard mode uses temporary files with SQL COPY commands (requires superuser privileges)
- Alternative `-s` flag enables non-superuser imports without temporary files

**Performance Features:**
- Index generation can be disabled via `-I` flag for faster imports
- Temporary directory specification via `-d` option allows optimization across multiple disk drives

**CSV Output Formatting:**
CSV files implement PostgreSQL escaping conventions, where backslashes, newlines, carriage returns, and commas are escaped with preceding backslashes, and NULL values display as `\\N`.
