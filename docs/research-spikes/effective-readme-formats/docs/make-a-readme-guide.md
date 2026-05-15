# Make a README Guide
- **Source**: https://www.makeareadme.com/
- **Retrieved**: 2026-05-15

## Core Purpose
A README is described as "a text file that introduces and explains a project" containing "information that is commonly required to understand what the project is about." The guide emphasizes that READMEs help answer questions about installation, usage, and collaboration opportunities.

## Recommended Sections and Advice

### Name
The guide advises choosing "a self-explaining name for your project" to immediately convey its purpose.

### Description
This section should "let people know what your project can do specifically" and "provide context and add a link to any reference visitors might be unfamiliar with." Optional subsections like Features or Background can enhance clarity. If alternatives exist, differentiate your project here.

### Badges
The guide suggests using visual indicators of project metadata, such as test status. It recommends using [Shields](http://shields.io/) or following service-specific badge instructions.

### Visuals
For appropriate projects, including "screenshots or even a video (you'll frequently see GIFs rather than actual videos)" helps users understand functionality. Tools like Asciinema offer sophisticated demonstrations.

### Installation
Step-by-step instructions should assume novice readers and "remove ambiguity and get people to using your project as quickly as possible." Include Requirements subsections when specific programming languages, operating systems, or manual dependencies apply.

### Usage
The guide recommends using "examples liberally, and show the expected output if you can," providing "the smallest example of usage that you can demonstrate" inline while linking to more complex examples.

### Support
"Tell people where they can go to for help" through issue trackers, chat rooms, email, or other channels.

### Roadmap
Document future release ideas to demonstrate ongoing development.

### Contributing
Clearly state openness to contributions and requirements for acceptance. Provide explicit setup instructions, including environment variables and scripts. Document linting commands and testing procedures to ensure quality. This section helps both external contributors and future maintainers.

### Authors and Acknowledgment
"Show your appreciation to those who have contributed to the project."

### License
"For open source projects, say how it is licensed." The guide directs readers to choosealicense.com for selection assistance.

### Project Status
Include notices if development has slowed or stopped, inviting forks or maintainers to continue the project.

## Template Example

The guide provides an editable Markdown template for "Foobar," a Python pluralization library:

```markdown
# Foobar
Foobar is a Python library for dealing with word pluralization.

## Installation
Use the package manager [pip](https://pip.pypa.io/en/stable/) to install foobar.

```bash
pip install foobar
```

## Usage
```python
import foobar
foobar.pluralize('word')      # returns 'words'
foobar.pluralize('goose')     # returns 'geese'
foobar.singularize('phenomena') # returns 'phenomenon'
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change. Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)
```

## File Naming and Format

The guide specifies that files should be named "README.md (or a different file extension if you choose to use a non-Markdown file format)," with traditional uppercase naming for prominence. Markdown is presented as the most common format.

## Key Reasoning

The guide emphasizes that "too long is better than too short" for README length, suggesting additional documentation elsewhere rather than cutting content. It notes that README sections should be customized: "Not all of the suggestions here will make sense for every project."

## Additional Documentation Resources

Beyond READMEs, the guide recommends wikis and documentation sites using tools like Daux.io, Docusaurus, GitBook, MkDocs, Read the Docs, ReadMe, Slate, and Docsify. Related files mentioned include CONTRIBUTING.md, CHANGELOG.md, issue templates, and pull request templates.
