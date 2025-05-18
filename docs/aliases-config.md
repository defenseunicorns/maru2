# Maru2 Aliases Configuration

This document describes how to configure and use package URL aliases in Maru2.

## Overview

Maru2 supports the use of aliases to simplify package URL references. Aliases allow you to create shorthand references for commonly used package types and repositories.

## Configuration File Location

By default, Maru2 looks for the aliases configuration file at:

```
~/.maru2/aliases.yaml
```

## Configuration Format

The aliases configuration file uses YAML format with the following structure:

```yaml
aliases:
  alias_name:
    type: package_type
    base: base_url
    token-from-env: env_variable_name
```

Where:

- `alias_name`: A short name you want to use as an alias
- `type`: The actual package URL type (github, gitlab, etc.)
- `base`: (Optional) Base URL for the repository (useful for self-hosted instances)
- `token-from-env`: (Optional) Environment variable name containing an access token

## Example Configuration

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

## Usage

Once configured, you can use aliases in package URLs:

```
pkg:gh/owner/repo@main#path/to/file.yaml
```

Instead of:

```
pkg:github/owner/repo@main#path/to/file.yaml
```

## Alias Resolution

When Maru2 encounters an alias in a package URL:

1. It looks up the alias in the configuration
2. Replaces the alias with the actual package type
3. Adds any configured base URL or token information (if not already specified)
4. Preserves all other parts of the package URL (namespace, name, version, subpath)

## Creating a New Configuration

To create a new aliases configuration:

1. Create the directory if it doesn't exist:

   ```
   mkdir -p ~/.maru2
   ```

2. Create the aliases.yaml file:

   ```
   touch ~/.maru2/aliases.yaml
   ```

3. Edit the file with your preferred text editor and add your aliases following the format shown above.

## Overriding Base URLs

You can override the base URL specified in the alias configuration by including it directly in the package URL:

```
pkg:gl/owner/repo@main?base=https://my-gitlab.com#path/to/file.yaml
```

This will use `https://my-gitlab.com` instead of the base URL configured in the alias.
