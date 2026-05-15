<!-- Source: http://blog.gilliard.lol/2018/10/25/live-coding-tips.html -->
<!-- Retrieved: 2026-05-15 -->

# Live Coding in Presentations: A Comprehensive Guide

## Preparation Phase

Scripting and automation before presenting are essential. Key preparation includes:

- **Environment setup**: Boot virtual machines beforehand and install required tools in advance. "45 seconds are _just_ long enough for the audience to lose focus" during startup.
- **Automation**: Create scripts like `prepare-for-demo.sh` to reset files and configurations between runs, avoiding wasted time on repetitive tasks.
- **File organization**: Pre-create multiple versions of configuration files with descriptive names rather than building them during the presentation.
- **Structure clarity**: Maintain clear talk sections (even numbered ones) so audiences can follow the progression and understand where live coding fits within the overall narrative.

## Terminal Configuration

**Visual presentation** requires careful attention:

- Choose "black-on-white" color schemes for better projector visibility and contrast
- Test font sizes in advance; run to the back of the room to verify readability
- Watch for line wrapping issues and use backslashes to break lengthy commands
- Simplify the prompt display (e.g., `export PS1=$'conf-name:topic\n> '`) to preserve screen space

**Command efficiency** matters significantly:

- Use aliases (`alias k=kubectl`) to reduce typing errors under pressure
- Leverage `ctrl-r` for command history searches, especially with hashtag comments
- Apply `bat` or `vim` for syntax-highlighted file viewing instead of basic `cat`

## Switching and Multi-Monitor Challenges

When alternating between slides and terminal windows, consider:

- Using separate workspaces in window managers for smoother transitions
- Creating mostly-blank slides with transparent terminals overlaid for simultaneous reference
- Keeping visual action in the upper half of the screen since bottom portions disappear behind audience members
- Using `xrandr` scripts or `tmux` with multiple clients for dual-monitor complexity

## Contingency Planning

- Maintain humor when errors occur; ask audience members who spot mistakes
- Eliminate internet dependencies or download large files beforehand
- Keep backup video recordings (using tools like Asciinema) for catastrophic failures
