# Essential Sections for Better Documentation of a README Project

- **Source URL**: https://www.welcometothejungle.com/en/articles/btc-readme-documentation-best-practices
- **Retrieved**: 2026-05-15

---

## Historical Context

README files trace back to 1974 on PDP-10 mainframes. The format evolved through UNIX manual pages in the 1990s, which introduced standardized structures and visual formatting. GitHub's rise in the 2000s transformed READMEs into repository landing pages, with Markdown becoming the de facto standard for documentation markup.

## Core Audience Considerations

**Know Your Readers**: Different audiences — contributors, end users, and designers — have distinct needs and skill levels. A single README must address multiple personas without assuming prior knowledge.

The author recommends seeking feedback from colleagues: "What they see as good are highlights to me, while what they do not understand...show where things need to be improved."

## Structural Best Practices

### Formatting Fundamentals

Modern READMEs leverage HTML5 features within Markdown. Key formatting tools include:

- **Headlines** create visual hierarchy and scanability
- **Emphasis and lists** break up dense content
- **Interactive elements** like `<details>` tags enhance engagement
- **Tables** efficiently present multi-entry information
- **Code blocks** display sample outputs clearly

### Information Architecture

Effective organization follows a narrative arc: general information first, then deeper details, concluding with supplementary content. A **table of contents** provides navigation for non-linear readers.

Headlines should be action-oriented or question-based rather than generic labels. Compare "Description" (invisible to readers) versus "What is it for?" (directly addresses reader questions).

## Essential Sections

### 1. Description
A concise one-liner explaining the software's purpose, followed by context about inputs, outputs, and requirements. Avoid assuming readers understand your motivation or domain.

### 2. Usage
"Writing explicitly takes more effort," but this section demonstrates practical code examples and multiple use case variations. Show different user interfaces and invocation methods.

### 3. Installation
Straightforward setup instructions from prerequisite environment through ready-to-use status. Accommodate platform-specific variations. Complex installation processes increase abandonment risk.

### 4. API Documentation
For libraries and services, thoroughly document each public interface with:
- Intent and purpose
- Mandatory and optional parameters
- Object shapes and return types
- Illustrative examples

Link to dedicated API documentation websites for extensive reference material rather than bloating the README.

### 5. Contributors & Contributing
The "Contributors" section acknowledges people who shaped the project. The "Contribute" section explains the contribution process — though many projects use separate `CONTRIBUTING.md` files for detailed guidance.

### 6. Guides and Resources
Supplement the README with links to extended documentation, migration guides, conference presentations, and community forums without overwhelming the primary document.

### 7. License
Display the license name with a link to the full legal text in a dedicated `LICENSE` file.

## Tone and Voice Considerations

"Carefully choosing words is a thing," as evidenced by organizations like MailChimp and UK Digital Services. Avoid language suggesting obviousness — terms like "easy," "obviously," and "simple" risk making readers feel inadequate.

Authentic, human-centered writing builds trust. Readers should sense "real humans are behind the words and the code."

## Visual Communication Strategies

**Screenshots and GIFs** immediately clarify software purpose, especially for desktop applications. **Sample outputs** in code blocks demonstrate data structures without requiring readers to execute code. **Tables** efficiently present comparative information like platform support.

## Anti-Patterns to Avoid

- **Unexplained jargon** alienates newcomers
- **Vague section titles** lack scanability
- **Missing visual examples** force readers to imagine functionality
- **Overwhelming installation procedures** encourage abandonment
- **Dense, unformatted text** resists reading

## Practical Tools

Recommended resources for README creation:
- **Preview tools**: Markdown Preview Plus (Atom), VSCode extensions, vmd
- **Visual enhancement**: Carbon (code screenshots), Gifski (GIF optimization)
- **Documentation platforms**: ReadTheDocs, GitBook, Antora for extended content
- **Screencast capture**: Peek, Kap for animated demonstrations

## Key Takeaway

"All I expect from a README file is...to provide me with the content I need to get started and to obtain a result." Effective documentation balances brevity with comprehensiveness, clarity with technical accuracy, and professional structure with approachable tone — transforming a simple text file into a project's most powerful advocacy tool.
