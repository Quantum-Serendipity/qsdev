<!-- Source: https://kapeli.com/docsets -->
<!-- Retrieved: 2026-05-14 -->

# Dash Docset Format Specification

## Directory Structure

```
<docset name>.docset/
├── Contents/
│   ├── Info.plist
│   └── Resources/
│       ├── Documents/
│       │   └── [HTML documentation files]
│       └── docSet.dsidx
└── icon.png (optional)
```

## Info.plist Configuration

Key fields:
- **dashIndexFilePath**: Main page displayed when opening the docset, relative to Documents folder
- **DashDocSetFallbackURL**: Base URL for online redirection to live documentation
- **DashDocSetPlayURL**: URL to a playground for the language/framework
- **DashDocSetFamily**: Set to "dashtoc" to enable table of contents support
- **isJavaScriptEnabled**: Boolean to enable external JavaScript files
- **DashDocSetDefaultFTSEnabled**: Boolean to enable full-text search by default
- **DashDocSetFTSNotSupported**: Boolean to completely disable full-text search

## SQLite Database Schema

```sql
CREATE TABLE searchIndex(
    id INTEGER PRIMARY KEY, 
    name TEXT, 
    type TEXT, 
    path TEXT
);

CREATE UNIQUE INDEX anchor ON searchIndex (name, type, path);
```

Insert entries: `INSERT OR IGNORE INTO searchIndex(name, type, path) VALUES ('name', 'type', 'path');`

### Column Definitions

- **name**: The entry's display name (what Dash searches)
- **type**: Category of the entry (see supported types below)
- **path**: Relative path to HTML file (can include anchors with #) or HTTP URL

## Supported Entry Types (90+)

Annotation, Attribute, Binding, Builtin, Callback, Category, Class, Command, Component, Constant, Constructor, Define, Delegate, Diagram, Directive, Element, Entry, Enum, Environment, Error, Event, Exception, Extension, Field, File, Filter, Framework, Function, Global, Guide, Hook, Instance, Instruction, Interface, Keyword, Library, Literal, Macro, Method, Mixin, Modifier, Module, Namespace, Notation, Object, Operator, Option, Package, Parameter, Plugin, Procedure, Property, Protocol, Provider, Provisioner, Query, Record, Resource, Sample, Section, Service, Setting, Shortcut, Statement, Struct, Style, Subroutine, Tag, Test, Trait, Type, Union, Value, Variable, Word

## Table of Contents Support

Insert special anchor tags in HTML:
```html
<a name="//apple_ref/cpp/Entry Type/Entry Name" class="dashAnchor"></a>
```

Requires `DashDocSetFamily: dashtoc` in Info.plist.

## Distribution via Docset Feed

XML feed format:
```xml
<entry>
    <version>[version number]</version>
    <url>[URL to archived docset]</url>
</entry>
```

Archive: `tar --exclude='.DS_Store' -cvzf <docset name>.tgz <docset name>.docset`
