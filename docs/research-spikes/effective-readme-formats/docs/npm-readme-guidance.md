# npm Official README Guidance
- **Source**: https://docs.npmjs.com/about-package-readme-files/
- **Retrieved**: 2026-05-15

## Recommended Content

npm recommends that README.md files should include:

- "directions for _installing_, _configuring_, and _using_ the code" in your package
- "any other information a user may find helpful"

The documentation emphasizes that a README helps "developers find your package on npm and have a good experience using your code in their projects."

## Rendering Details

npm displays README files with specific technical handling:

- The README must be named `README.md` and use Markdown formatting
- Files are "rendered as GitHub Flavored Markdown via GitHub's API"
- The rendered preview appears on the package's npmjs.com page

## npm-Specific Constraints and Features

**Location requirement:** "An npm package README.md file must be in the root-level directory of the package."

**Update mechanism:** Changes to README content only appear when you publish a new package version using `npm version patch` followed by `npm publish`. The documentation notes that "The README.md file will only be updated on the package page when you publish a new version of your package."

This update requirement distinguishes npm's behavior from GitHub repositories, where README changes display immediately.
