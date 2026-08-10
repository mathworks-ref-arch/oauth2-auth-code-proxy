# Contributing to OAuth2 Token Proxy

Thank you for your interest in contributing.

## How to Contribute

1. Fork the repository.
2. Create a feature branch from `main`.
3. Make your changes, ensuring tests pass (`make test`) and lint is clean (`make lint`).
4. Submit a pull request with a clear description of the change and how it was tested.

## Requirements for Pull Requests

- All tests must pass with race detection enabled.
- Code coverage must remain at or above 80%.
- No lint violations (golangci-lint v2).
- Commit messages should describe *why*, not just *what*.

## Reporting Issues

Please open a GitHub issue with:
- Steps to reproduce
- Expected behavior
- Actual behavior
- Go version and OS

## Security Issues

Do **not** report security vulnerabilities via GitHub issues. See [SECURITY.md](SECURITY.md) for responsible disclosure instructions.

## License

By contributing, you agree that your contributions will be licensed under the same terms as this project. See [LICENSE.md](LICENSE.md).
