# Maru2 Configuration

This document describes how to configure Maru2 using the global configuration file.

## Configuration File Location

By default, Maru2 looks for the configuration file at:

```text
~/.maru2/config.yaml
```

and can also be configured via the [`--config`](./cli.md#config) flag.

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

3. Edit the file with your preferred text editor and add your configuration. The default configuration is shown below.

## Default Configuration

```yaml
schema-version: v0
fetch-policy: "if-not-present"
aliases: {}
```

[Fetch Policy](./cli.md#fetch-policy) and [Aliases](./syntax.md#package-url-aliases).

Note: aliases configured via the config file only affect `-f/--from` alias resolution.

## Future Configuration Options

The global configuration file is designed to be extensible. Future versions of Maru2 may add additional configuration options.
