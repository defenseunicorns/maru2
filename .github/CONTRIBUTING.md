# Contributing to `defenseunicorns/maru2`

> Read [the developers guide to Maru2](../docs/developing.md) for some good information to know when developing Maru2.

1. Install project dependencies and build all binaries.

   ```bash
   make
   ```

2. Create a new branch on your fork.

   ```bash
   git switch -c <branch>
   ```

3. Make your changes.

4. Run the tests, linters, and formatters.

   > run `make test ARGS="-w short=true` if you want to skip tests that require external network connections with authentication (GitHub/Gitlab/etc...)

   See [E2E testing README](../testdata/README.md) for more info on how the E2E tests run.

   ```bash
   make all lint test
   ```

5. Commit your changes.

   > Set up your Git config to sign all commits. [Here's some documentation on how to set it up](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits). You won't be able to merge your PR if you have any unverified commits.

   ```bash
   git commit -m "feat: add new feature"
   ```

6. Push your changes to your fork.

   ```bash
   git push --set-upstream <fork> <branch>
   ```

7. Open a pull request. The title must follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) format (`fix:`, `feat:`, `chore:`, `docs:`, etc...). For example:

   ```bash
   feat: add new feature
   ```

   > Use `wip:` if you are unsure about the changes and want feedback about the scope of the PR.

8. Once your pull request is approved, your changes will be available in the next release. Thanks!
