# Contributing guide

## Welcome contributions

Thank you for your interest in the E-SP-Line2 project! We welcome every kind of contribution, including code, documentation, issue reports, and feature suggestions.

## Contribution workflow

### 1. Find a problem or a feature you want to add

- Check [Issues](https://github.com/E-SPower/E-SP-Line2/issues) for an existing discussion
- If there is none, create a new issue describing the problem or feature

### 2. Start development

- Fork the project to your GitHub account
- Clone it locally:
  ```bash
  git clone https://github.com/your-username/E-SP-Line2.git
  cd E-SP-Line2
  ```
- Create a feature branch:
  ```bash
  git checkout -b feature/your-feature-name
  ```

### 3. Development conventions

#### Code style

- **Go code**: Follow [Effective Go](https://go.dev/doc/effective-go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- **Python code**: Follow [PEP 8](https://peps.python.org/pep-0008/)
- **TypeScript/React**: Follow the project's ESLint configuration

#### Commit messages

Use Conventional Commits:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `style`: Code formatting (does not affect how the code runs)
- `refactor`: Code refactoring
- `test`: Add tests
- `chore`: Build process or auxiliary tooling changes

**Example**:
```
feat(adapter): add support for Pinduoduo platform

- Implement Pinduoduo message adapter
- Add product card message type
- Add unit tests

Closes #123
```

### 4. Submit a pull request

- Make sure the code passes all tests
- Update the relevant documentation
- Create a pull request describing the changes

## Development environment setup

### Backend development

```bash
# Install Go dependencies
go mod download

# Run tests (CGO_ENABLED=1 — the sqlite driver is cgo)
CGO_ENABLED=1 go test ./...

# Start the development server
go run main.go
```

### Frontend development

```bash
# Install dependencies
cd web
npm install

# Start the development server
npm run dev

# Build a production version
npm run build
```

### Python adapter development

```bash
# Create a virtual environment
cd adapters/myplatform
python -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Run tests
pytest
```

## Code review

Every pull request goes through code review. Reviewers focus on:

- Code quality and style
- Test coverage
- Documentation completeness
- Performance impact
- Security

CI (`.github/workflows/ci.yml`) runs the mechanical part of this list. The full reviewer checklist, merge gate and exception process is [pr-qa.md](pr-qa.md). Backend tests need `CGO_ENABLED=1` and gcc, or the sqlite driver fails against a stub.

## Reporting bugs

### Bug report template

```markdown
**Description**
Describe the bug clearly and concisely.

**Steps to reproduce**
1. Run '...'
2. Click '...'
3. See the error

**Expected behavior**
Describe what you expected to happen.

**Actual behavior**
Describe what actually happened.

**Environment**
- OS: [e.g. Ubuntu 22.04]
- Go version: [e.g. 1.22]
- Node version: [e.g. 20.11] (must satisfy `web/package.json` `engines`)

**Logs**
```
Paste the relevant logs
```

**Screenshots**
Add screenshots if applicable.
```

## Feature suggestions

### Feature suggestion template

```markdown
**Problem description**
Describe the problem or need clearly and concisely.

**Proposed solution**
Describe the solution you suggest.

**Alternatives**
Describe the alternatives you considered.

**Additional information**
Add any other relevant information.
```

## Documentation contributions

Documentation matters as much as code. When contributing documentation:

- Use clear Markdown formatting
- Provide code examples
- Keep the language concise
- Check spelling and grammar

## Testing

### Writing tests

- Write unit tests for new features
- Write regression tests for bug fixes
- Keep test coverage above 80%

### Running tests

```bash
# Backend tests (CGO_ENABLED=1 is mandatory, see pr-qa.md)
CGO_ENABLED=1 go test ./... -v

# Frontend: no `npm test` script exists yet (no runner installed).
# The gates that do exist, and that CI runs:
cd web
npx tsc --noEmit
npm run lint
npm run build

# Python adapter tests
cd adapters/myplatform
pytest
```

Front-end behaviour is verified by hand for now: `web/package.json` has no `test`
script and no runner is installed, so a page change is verified by building it and
walking the route. Adding Vitest and an `npm test` script is its own PR — not a
prerequisite for merging a UI change, but a review may ask for one on logic-heavy
code (see [pr-qa.md](pr-qa.md) §3).

## Release process

### Version numbers

Follow Semantic Versioning:

- `MAJOR.MINOR.PATCH`
- `MAJOR`: Incompatible API changes
- `MINOR`: Backward-compatible feature additions
- `PATCH`: Backward-compatible bug fixes

### Release checklist

- [ ] Update the version number
- [ ] Update CHANGELOG
- [ ] Run all tests
- [ ] Build a production version
- [ ] Create a Git tag
- [ ] Publish to the package manager

## Code of conduct

### Our pledge

To create an open and friendly environment, we pledge to:

- Use friendly and inclusive language
- Respect different viewpoints and experiences
- Accept constructive criticism gracefully
- Focus on what is best for the community
- Show empathy toward other community members

### Unacceptable behavior

- Use of sexualized language or imagery
- Personal attacks or insulting comments
- Public or private harassment
- Publishing others' information without permission
- Other unethical or unprofessional behavior

## Getting help

If you need help while contributing:

- Read the [documentation](./)
- Ask in an issue
- Contact the project maintainers

## Acknowledgments

Thank you to all contributors for helping improve the E-SP-Line2 project!
