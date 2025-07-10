# Maru2 Configuration

This document describes how to configure Maru2 using the global configuration file.

## Configuration File Location

By default, Maru2 looks for the configuration file at:

```
~/.maru2/config.yaml
```

## Creating a New Configuration

To create a new configuration:

1. Create the directory if it doesn't exist:

   ```sh
   mkdir -p ~/.maru2
   ```

2. Create the config.yaml file:

   ```sh
   touch ~/.maru2/config.yaml
   ```

3. Edit the file with your preferred text editor and add your configuration following the format shown below.

## Aliases Configuration

Aliases allow you to create shorthand references for commonly used package types and repositories, simplifying package URL references in your workflows.

### Aliases Format

```yaml
aliases:
  alias_name:
    type: package_type
    base: base_url
    token-from-env: env_variable_name
```

Where:

- `alias_name`: A short name you want to use as an alias
- `type`: The actual package URL type (github, gitlab, etc.) - this is required
- `base`: (Optional) Base URL for the repository (useful for self-hosted instances)
- `token-from-env`: (Optional) Environment variable name containing an access token. Environment variable names must start with a letter or underscore, and can contain letters, numbers, and underscores (e.g., `MY_ENV_VAR`, `_ANOTHER_VAR`).

### Example Aliases Configuration

```yaml
aliases:
  gh:
    type: github
  gl:
    type: gitlab
    base: https://gitlab.example.com
  custom:
    type: github
    token-from-env: GITHUB_TOKEN
```

### Using Aliases

Once configured, you can use aliases in package URLs:

```yaml
pkg:gh/owner/repo@main#path/to/file.yaml # Using the 'gh' alias
```

Instead of the full type name:

```yaml
pkg:github/owner/repo@main#path/to/file.yaml
```

### Alias Resolution

When Maru2 encounters an alias in a package URL:

1. It looks up the alias in the configuration
2. Replaces the alias with the actual package type
3. Adds any configured base URL or token information (if not already specified)
4. Preserves all other parts of the package URL (namespace, name, version, subpath)

### Overriding Base URLs

You can override the base URL specified in the alias configuration by including it directly in the package URL:

```yaml
pkg:gl/owner/repo@main?base=https://my-gitlab.com#path/to/file.yaml
```

This will use `https://my-gitlab.com` instead of the base URL configured in the alias.

### Token Authentication for Private Repositories

For private repositories, you can configure authentication tokens using the `token-from-env` property:

```yaml
aliases:
  private:
    type: github
    token-from-env: GITHUB_TOKEN
```

With this configuration, Maru2 will read the token from the specified environment variable and use it for authentication when accessing repositories through this alias.

## Future Configuration Options

The global configuration file is designed to be extensible. Future versions of Maru2 may add additional configuration options beyond aliases.
