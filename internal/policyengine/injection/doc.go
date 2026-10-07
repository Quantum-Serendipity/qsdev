// Package injection detects high-signal prompt-injection indicators in text
// an agent is about to read, such as untrusted MCP tool output.
//
// Scan reports which indicator classes are present, once per class:
// instruction-override phrases, forged qsdev frame tags, chat-template role
// markers, and Unicode TAG characters outside valid emoji tag sequences.
// Markers that are ordinary in web pages and tool output ("{{", "${", "</",
// "<script", "javascript:", bare "system:" keys, base64 blocks, BiDi
// isolates) are deliberately not indicators: they are template or XSS syntax
// irrelevant to a model, and flagging them would bury the real signal.
//
// Every phrase matches as a plain lowercase substring, so nothing the base
// substring checks caught is lost. Normalized views only add matches: a view
// with every run of non-alphanumeric runes collapsed to one space, the same
// with invisible runes deleted first, each decoded string token of a JSON
// document, and the letters hidden in Unicode TAG runes. In the collapsed
// views a phrase matches only as whole words, and a phrase with leading or
// trailing punctuation ("/etc/") is not normalized at all, so normalizing
// never shrinks a phrase into a fragment of an unrelated word. Every view is
// built in one linear pass.
package injection
