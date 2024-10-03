# Contributing
Thank you for considering contributing to this project! Any kind of contributions are more than welcome.

## Ways to contribute
There are many ways to contribute to our project, either by:

- Reporting a bug
- Submitting a fix
- Proposing new features
- Implementing features
- Updating/improving documentation

### Report bugs
We are using GitHub Issues for reporting and tracking bugs.
If you find a bug, please create a new issue using the Bug Report issue template.

Make sure to include any helpful information in your bug report:
- a short summary and/or background
- detailed steps how to reproduce the issue
- expected and actual behavior
- suggestion on how to resolve the issue, if possible
- provide sample code, if possible

A detailed bug report will help to speed up the process of fixing the issue.

#### Security bugs
If you find any security related bugs, please **do not** post them as a new issue, but rather send the bug report by e-mail.

### Fix bugs
Look through the GitHub issues for bugs. Any open issue is open to whomever wants to contribute to its resolution.
If you find a bug and would like to fix it yourself, please raise an issue before you start any development.

### Suggesting new features
To suggest new features, plese create a new issue using the Feature Request issue template.

When proposing a new feature:
- Provide a detailed explanation on the identified problem and the proposed solution.
- Keep the scope as narrow as possible to make it easier to implement.

### Implement features
Look through the GitHub issues labeled "kind:feature" for features.
Any unassigned feature request issue is open to whomever wants to contribute to its implementation.

### Improve documentation
If you find areas of improvement for documentation, or you consider that certain parts are documented as well, feel free to contribute.

## Code Contribution workflow
Pull Requests are the best way to propose changes to the codebase. We are following the "fork-and-pull" Git workflow.
In short, these are steps that need to be executed:

1. Fork the repo on GitHub
2. Clone the project to your own machine
3. Commit changes to your own branch (we recommend [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages)
4. Push your work back up to your fork
5. Submit a Pull Request so that we can review your changes

Things to note:
- Before creating a Pull Request, merge the latest from "upstream" and check that there are no conflicts between your branch and the base branch.
- When creating a Pull Request, please provide sufficient information by filling out the generated Pull Request template.
- Provide a clear description of the issue that you want to resolve in the Pull Request message.
- If your Pull Request solves an existing issue, [link the issue in the Pull Request](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue).
- Please be patient while the maintainers review your Pull Request and append the requested changes before it can be approved.

## Coding style
We are using a static code analyzer as part of our CI pipeline, which also checks code style rules are being followed.
Please format your code using *goimports* and check the configuration of [.golangci.yaml](.golangci.yaml).
Additionally, read the [Dataphos Go Style Guide](STYLEGUIDE.md).

## Commit Message Guidelines
We follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) convention to maintain a clean and meaningful commit history. Please ensure that your commit messages follow this format. This helps automate our release process and makes it easier to understand the changes in the project.

To make this easier, you can setup a pre-commit hook that will validate your commit messages before submission. We recommend installing the hook to avoid any issues when committing changes.

Setup the hook with the pre-commit framework by creating the [.pre-commit-config.yaml](https://github.com/qoomon/git-conventional-commits?tab=readme-ov-file#setup-with-the-pre-commit-framework) file and the [git-conventional-commits.yaml](https://github.com/qoomon/git-conventional-commits?tab=readme-ov-file#config-file) config file in the root directory of your cloned repository. Replace the `<RELEASE_TAG>` with the latest release version of git-conventional-commits (e.g. `v2.6.7`).

Install the pre-commit framework: `pip install pre-commit`

Install the commit-msg hook: `pre-commit install -t commit-msg`

## License
By contributing, you agree that your contributions will be licensed under Apache License 2.0.

## Community
You can get in touch with us by e-mail: contributors@syntio.net
