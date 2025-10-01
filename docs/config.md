# Maru2 configuration

This document describes how to configure Maru2 using the global configuration file.

## Configuration file location

Maru2 loads configuration in priority order:

1. `--config` flag (highest priority)
2. `MARU2_CONFIG` environment variable
3. `~/.maru2/config.yaml` (default)

```sh
maru2 --config custom.yaml        # flag
MARU2_CONFIG=custom.yaml maru2    # env var
maru2                             # default
```

## Creating a new configuration

To create a new global configuration:

1. Create the directory if it doesn't exist:

   ```sh
   mkdir -p ~/.maru2
   ```

2. Create the config.yaml file:

   ```sh
   touch ~/.maru2/config.yaml
   ```

3. Edit the file with your preferred text editor and add your configuration. The default configuration is as follows.

## Default configuration

```yaml
schema-version: v0
fetch-policy: "if-not-present"
aliases: {}
```

[Fetch Policy](./cli.md#fetch-policy) and [Aliases](./syntax.md#package-url-aliases).

Note: aliases defined in the global configuration file apply only to the `-f`/`--from` flag for resolving the main workflow file. They're not available for `uses:` steps within a workflow. For aliases used in `uses:`, define them within the workflow file's `aliases` block.

## Future configuration options

The global configuration file is extensible. Future versions of Maru2 may add additional configuration options.
