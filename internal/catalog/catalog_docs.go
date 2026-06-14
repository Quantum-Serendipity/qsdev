package catalog

// --- Docs Corpus accessors ---

// DocsCorpus returns the documentation corpus configuration.
func (c *Catalog) DocsCorpus() DocsCorpusConfig {
	return c.docsCorpus
}

// DevDocsBaseURL returns the base URL for DevDocs API downloads.
func (c *Catalog) DevDocsBaseURL() string {
	if c.docsCorpus.DevDocsBaseURL != "" {
		return c.docsCorpus.DevDocsBaseURL
	}
	return "https://documents.devdocs.io"
}

// DevDocsSlugs returns the language-to-slug mapping for DevDocs.
func (c *Catalog) DevDocsSlugs() map[string][]string {
	return c.docsCorpus.DevDocsSlugs
}

// ZIMArchives returns the configured ZIM archive entries.
func (c *Catalog) ZIMArchives() []ZIMArchiveDef {
	return c.docsCorpus.ZIMArchives
}
