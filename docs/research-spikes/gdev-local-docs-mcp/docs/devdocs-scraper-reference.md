<!-- Source: https://raw.githubusercontent.com/freeCodeCamp/devdocs/main/docs/scraper-reference.md -->
<!-- Retrieved: 2026-05-14 -->

# DevDocs Scraper Reference (Full Raw Content)

**Table of contents:**

* [Overview](#overview)
* [Configuration](#configuration)
  - [Attributes](#attributes)
  - [Filter stacks](#filter-stacks)
  - [Filter options](#filter-options)

## Overview

Starting from a root URL, scrapers recursively follow links that match a set of rules, passing each valid response through a chain of filters before writing the file on the local filesystem. They also create an index of the pages' metadata (determined by one filter), which is dumped into a JSON file at the end.

Scrapers rely on the following libraries:

* [Typhoeus](https://github.com/typhoeus/typhoeus) for making HTTP requests
* [HTML::Pipeline](https://github.com/jch/html-pipeline) for applying filters
* [Nokogiri](http://nokogiri.org/) for parsing HTML

There are currently two kinds of scrapers: [`UrlScraper`](https://github.com/freeCodeCamp/devdocs/blob/main/lib/docs/core/scrapers/url_scraper.rb) which downloads files via HTTP and [`FileScraper`](https://github.com/freeCodeCamp/devdocs/blob/main/lib/docs/core/scrapers/file_scraper.rb) which reads them from the local filesystem. They function almost identically (both use URLs), except that `FileScraper` substitutes the base URL with a local path before reading a file. `FileScraper` uses the placeholder `localhost` base URL by default and includes a filter to remove any URL pointing to it at the end.

To be processed, a response must meet the following requirements:

* 200 status code
* HTML content type
* effective URL (after redirection) contained in the base URL (explained below)

(`FileScraper` only checks if the file exists and is not empty.)

Each URL is requested only once (case-insensitive).

## Configuration

Configuration is done via class attributes and divided into three main categories:

* [Attributes](#attributes) — essential information such as name, version, URL, etc.
* [Filter stacks](#filter-stacks) — the list of filters that will be applied to each page.
* [Filter options](#filter-options) — the options passed to said filters.

**Note:** scrapers are located in the [`lib/docs/scrapers`](https://github.com/freeCodeCamp/devdocs/tree/main/lib/docs/scrapers/) directory. The class's name must be the [CamelCase](http://api.rubyonrails.org/classes/String.html#method-i-camelize) equivalent of the filename.

### Attributes

* `name` [String] - Must be unique. Defaults to the class's name.
* `slug` [String] - Must be unique, lowercase, and not include dashes (underscores are ok). Defaults to `name` lowercased.
* `type` [String] **(required, inherited)** - Defines the CSS class name (`_[type]`) and custom JavaScript class (`app.views.[Type]Page`) that will be added/loaded on each page. Documentations sharing a similar structure should use the same `type` to avoid duplicating the CSS and JS. Must include lowercase letters only.
* `release` [String] **(required)** - The version of the software at the time the scraper was last run. This is only informational.
* `base_url` [String] **(required in UrlScraper)** - The documents' location. Only URLs inside the `base_url` will be scraped. "inside" more or less means "starting with" except that `/docs` is outside `/doc` (but `/doc/` is inside). Defaults to `localhost` in `FileScraper`.
* `base_urls` [Array] **(requires MultipleBaseUrls module)** - Multiple documentation locations.
* `root_path` [String] **(inherited)** - The path from the `base_url` of the root URL.
* `initial_paths` [Array] **(inherited)** - A list of paths to add to the initial queue. Defaults to `[]`.
* `dir` [String] **(required, FileScraper only)** - The absolute path where the files are located on the local filesystem.
* `params` [Hash] **(inherited, UrlScraper only)** - Query string parameters to append to every URL. Defaults to `{}`.
* `abstract` [Boolean] - Make the scraper abstract / not runnable. Defaults to `false`.

### Filter stacks

Each scraper has two filter stacks: `html_filters` and `text_filters`. They are combined into a pipeline (using the HTML::Pipeline library) which causes each filter to hand its output to the next filter's input.

HTML filters are executed first and manipulate a parsed version of the document (a Nokogiri node object), whereas text filters manipulate the document as a string. This separation avoids parsing the document multiple times.

Filter stacks are like sorted sets. They can modified using:

```ruby
push(*names)                 # append one or more filters at the end
insert_before(index, *names) # insert one filter before another
insert_after(index, *names)  # insert one filter after another
replace(index, name)         # replace one filter with another
```

Default `html_filters`:
* `ContainerFilter` — changes the root node of the document
* `CleanHtmlFilter` — removes HTML comments, `<script>`, `<style>`, etc.
* `NormalizeUrlsFilter` — replaces all URLs with their fully qualified counterpart
* `InternalUrlsFilter` — detects internal URLs and replaces them with relative counterpart
* `NormalizePathsFilter` — makes the internal paths consistent
* `CleanLocalUrlsFilter` — removes links, iframes and images pointing to localhost (FileScraper only)

Default `text_filters`:
* `InnerHtmlFilter` — converts the document to a string
* `CleanTextFilter` — removes empty nodes
* `AttributionFilter` — appends the license info and link to the original document

Additionally:
* `TitleFilter` is a core HTML filter, disabled by default, which prepends the document with a title (`<h1>`).
* `EntriesFilter` is an abstract HTML filter that each scraper must implement, responsible for extracting the page's metadata.

### Filter options

The filter options are stored in the `options` Hash. The Hash is inheritable (a recursive copy) and empty by default.

**ContainerFilter**: `:container` [String or Proc] - A CSS selector of the container element. Default is `<body>`.

**NormalizeUrlsFilter**:
- `:replace_urls` [Hash] - Replaces all instances of a URL with another.
- `:replace_paths` [Hash] - Replaces all instances of a sub-path with another.
- `:fix_urls` [Proc] - Called with each URL for custom modification.

**InternalUrlsFilter**:
- `:skip_links` [Boolean or Proc] - If false, does not convert or follow any internal URL.
- `:follow_links` [Proc] - Controls whether to add internal URLs to the queue.
- `:trailing_slash` [Boolean] - Adds/removes trailing slashes.
- `:skip` [Array] - Ignores internal URLs whose sub-paths are in the Array.
- `:skip_patterns` [Array] - Ignores internal URLs matching Regexps.
- `:only` [Array] - Only allows internal URLs whose sub-paths are in the Array.
- `:only_patterns` [Array] - Only allows internal URLs matching Regexps.

**AttributionFilter**: `:attribution` [String] **(required)** - HTML string with copyright/license info.

**TitleFilter**:
- `:title` [String or Boolean or Proc] - Controls title generation.
- `:root_title` [String or Boolean] - Overrides `:title` for root page only.

### Processing responses before filters

* `process_response?(response)` - Determine whether a response should be processed. Returns false to drop.
* `parse(response)` - Parse HTTP/File response, convert to Nokogiri document. Override to modify HTML before Nokogiri.

## Keeping scrapers up-to-date

Override `get_latest_version(opts)` to track documentation updates.

### Utility Methods

**HTTP Methods:**
- `fetch(url, opts)` - GET request returning body
- `fetch_doc(url, opts)` - GET request returning Nokogiri document
- `fetch_json(url, opts)` - GET request returning parsed JSON

**Package Repository:**
- `get_npm_version(package, opts)` - Latest npm package version

**GitHub:**
- `get_latest_github_release(owner, repo, opts)` - Latest release tag
- `get_github_tags(owner, repo, opts)` - List of repository tags
- `get_github_file_contents(owner, repo, path, opts)` - File contents
- `get_latest_github_commit_date(owner, repo, opts)` - Most recent commit date

**GitLab:**
- `get_gitlab_tags(hostname, group, project, opts)` - Repository tags
